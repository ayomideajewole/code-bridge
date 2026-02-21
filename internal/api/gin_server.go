package api

import (
	"code-bridge/internal/services"
	"code-bridge/internal/sse"
	"code-bridge/pkg/types"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GinServer struct {
	router   *gin.Engine
	logger   *zap.Logger
	services *services.Services
	sseHub   *sse.Hub
}

func NewGinServer(logger *zap.Logger, services *services.Services) *GinServer {
	router := gin.Default()
	router.Use(GinLogger(logger))

	// Initialize SSE Hub
	sseHub := sse.NewHub()
	go sseHub.Run()

	server := &GinServer{
		router:   router,
		logger:   logger,
		services: services,
		sseHub:   sseHub,
	}
	server.SetupRoutes()
	return server
}

// GetRouter returns the Gin router
func (s *GinServer) GetRouter() *gin.Engine {
	return s.router
}

func (s *GinServer) SetupRoutes() {
	s.router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Serve web interface with no-cache headers
	s.router.GET("/web", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.File("./web/index.html")
	})

	// Serve static files with no-cache headers
	s.router.GET("/static/*filepath", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.File("./web" + c.Request.URL.Path)
	})

	s.router.GET("/health", s.HealthCheck)
	s.router.POST("/translate", s.TranslateCode)
	s.router.GET("/translate/stream/:id", s.StreamHandler)
}

// GinLogger returns a gin middleware for logging using zap
func GinLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Check if the API server is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (s *GinServer) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "codebridge-api",
	})
}

// TranslateCode handles code translation requests with Server-Sent Events
// @Summary Translate code from one language to another
// @Description Translates code using AI with streaming response via SSE
// @Tags translation
// @Accept json
// @Produce text/event-stream
// @Param request body types.TranslateRequest true "Translation request"
// @Success 200 {string} string "SSE stream"
// @Router /translate [post]
//func (s *GinServer) TranslateCode(c *gin.Context) {
//	var req types.TranslateRequest
//	if err := c.ShouldBindJSON(&req); err != nil {
//		c.JSON(400, gin.H{"error": err.Error()})
//		return
//	}
//
//	// Set headers for SSE
//	c.Header("Content-Type", "text/event-stream")
//	c.Header("Cache-Control", "no-cache")
//	c.Header("Connection", "keep-alive")
//	c.Header("X-Accel-Buffering", "no")
//
//	// Get the response writer
//	w := c.Writer
//
//	s.logger.Info("translation request",
//		zap.String("source_language", req.SourceLanguage),
//		zap.String("target_language", req.TargetLanguage),
//		zap.Int("code_length", len(req.Code)),
//	)
//
//	// Simulate streaming translation (placeholder for OpenAI/Gemini API)
//	// In production, this would call the actual AI API
//	translatedCode := s.simulateTranslation(req, w)
//
//	// Send final event
//	fmt.Fprintf(w, "data: %s\n\n", translatedCode)
//	w.Flush()
//
//	s.logger.Info("translation completed")
//}

// TranslateCode handles code translation requests with Server-Sent Events
// @Summary Translate code from one language to another
// @Description Translates code using AI with streaming response via SSE
// @Tags translation
// @Accept json
// @Produce text/event-stream
// @Param request body types.TranslateRequest true "Translation request"
// @Success 200 {string} string "SSE stream"
// @Router /translate [post]
func (s *GinServer) TranslateCode(c *gin.Context) {
	var req types.TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("translation request",
		zap.String("source_language", req.SourceLanguage),
		zap.String("target_language", req.TargetLanguage),
		zap.Int("code_length", len(req.Code)),
	)

	// create job id
	id := fmt.Sprintf("job-%d", time.Now().UnixNano())

	// create channel for streaming
	s.sseHub.Create(id)

	s.logger.Info("translation job created", zap.String("id", id))
	c.JSON(http.StatusAccepted, gin.H{"id": id})

	// call translator in background
	go func() {
		// Use a timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		time.Sleep(100 * time.Millisecond)

		s.logger.Info("starting translation", zap.String("id", id))

		// translator will push messages to hub via callback
		er := s.services.CodeTranslatorService.TranslateCode(ctx, req.Code, req.SourceLanguage, req.TargetLanguage, func(chunk string) error {
			s.logger.Debug("sending chunk", zap.String("id", id), zap.Int("chunk_size", len(chunk)))
			return s.sseHub.Send(id, chunk)
		})
		if er != nil {
			s.logger.Error("translation error", zap.String("id", id), zap.Error(er))
			_ = s.sseHub.Send(id, fmt.Sprintf("ERROR: %v", er))
		}
		// Always signal end, even on error
		s.logger.Info("translation finished, sending end signal", zap.String("id", id))
		_ = s.sseHub.Send(id, "[DONE]")
		s.logger.Info("translation completed", zap.String("id", id))
	}()
}

// StreamHandler attaches client to SSE stream
func (s *GinServer) StreamHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	s.logger.Info("client connecting to stream", zap.String("id", id))

	client := s.sseHub.AddClient(id)
	defer func() {
		s.logger.Info("client disconnecting from stream", zap.String("id", id))
		s.sseHub.RemoveClient(id, client)
	}()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.logger.Error("streaming not supported")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	// Send initial connection message to establish the stream
	fmt.Fprintf(c.Writer, ": connected\n\n")
	flusher.Flush()

	s.logger.Info("stream established", zap.String("id", id))

	// send existing backlog (if any)
	for {
		select {
		case msg, ok := <-client.Ch:
			if !ok {
				s.logger.Info("client channel closed", zap.String("id", id))
				return
			}

			// Log what we're sending
			s.logger.Debug("sending message to client",
				zap.String("id", id),
				zap.String("msg_preview", msg[:min(len(msg), 50)]))

			// Send the message as-is (including [DONE])
			fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
			flusher.Flush()

			// Check if this is the end signal
			if msg == "[DONE]" {
				s.logger.Info("stream end signal sent to client", zap.String("id", id))
				return
			}
		case <-c.Request.Context().Done():
			s.logger.Info("client context cancelled", zap.String("id", id))
			return
		}
	}
}

// simulateTranslation simulates streaming translation
// This is a placeholder - replace with actual OpenAI/Gemini API call
func (s *GinServer) simulateTranslation(req types.TranslateRequest, w io.Writer) string {
	chunks := []string{
		"// Translating code to " + req.TargetLanguage + "...\n",
		"// Processing...\n",
		"// Generated code:\n\n",
	}

	// Stream chunks with delay to simulate AI response
	for _, chunk := range chunks {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Return placeholder translated code
	return fmt.Sprintf("// TODO: Integrate with OpenAI/Gemini API\n// Original code length: %d\n// Target language: %s\n\nfunction placeholder() {\n  return 'Translation pending API integration';\n}", len(req.Code), req.TargetLanguage)
}

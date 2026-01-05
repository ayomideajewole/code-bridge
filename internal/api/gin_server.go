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

	s.router.StaticFile("/web", "./web/index.html")
	s.router.Static("/static", "web/static")

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

	// Set headers for SSE
	//c.Header("Content-Type", "text/event-stream")
	//c.Header("Cache-Control", "no-cache")
	//c.Header("Connection", "keep-alive")
	//c.Header("X-Accel-Buffering", "no")

	// Get the response writer
	//w := c.Writer

	s.logger.Info("translation request",
		zap.String("source_language", req.SourceLanguage),
		zap.String("target_language", req.TargetLanguage),
		zap.Int("code_length", len(req.Code)),
	)

	// create job id
	id := fmt.Sprintf("job-%d", time.Now().UnixNano())

	// create channel for streaming
	s.sseHub.Create(id)

	// call translator in background
	go func() {
		ctx := context.Background()
		// translator will push messages to hub via callback
		er := s.services.CodeTranslatorService.TranslateCode(ctx, req.Code, req.SourceLanguage, req.TargetLanguage, func(chunk string) error {
			return s.sseHub.Send(id, chunk)
		})
		if er != nil {
			_ = s.sseHub.Send(id, fmt.Sprintf("ERROR: %v", er))
		}
		// signal end
		_ = s.sseHub.Send(id, "__STREAM_END__")
	}()

	// Simulate streaming translation (placeholder for OpenAI/Gemini API)
	// In production, this would call the actual AI API
	//translatedCode := s.simulateTranslation(req, w)
	//
	//// Send final event
	//fmt.Fprintf(w, "data: %s\n\n", translatedCode)
	//w.Flush()
	c.JSON(http.StatusAccepted, gin.H{"id": id})
	s.logger.Info("translation completed", zap.String("id", id))
}

// StreamHandler attaches client to SSE stream
func (s *GinServer) StreamHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	client := s.sseHub.AddClient(id)
	defer s.sseHub.RemoveClient(id, client)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	// send existing backlog (if any)
	for {
		select {
		case msg, ok := <-client.Ch:
			if !ok {
				return
			}
			if msg == "__STREAM_END__" {
				fmt.Fprintf(c.Writer, "data: %s\n\n", "[DONE]")
				flusher.Flush()
				return
			}
			// write SSE event
			fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
			flusher.Flush()
		case <-c.Request.Context().Done():
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

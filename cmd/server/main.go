package main

import (
	"code-bridge/internal/api"
	"code-bridge/internal/code_translator"
	"code-bridge/internal/services"
	"code-bridge/internal/translator_provider"
	"code-bridge/pkg/database"
	"code-bridge/pkg/types"
	"context"
	"fmt"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Load application configuration from environment variables
	globalConfig, err := types.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize logger with human-readable timestamps
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.TimeKey = "time"
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logLevel := zap.InfoLevel
	if globalConfig.Server.LogLevel != "" {
		if err := logLevel.UnmarshalText([]byte(globalConfig.Server.LogLevel)); err != nil {
			logLevel = zap.InfoLevel
		}
	}
	logConfig.Level = zap.NewAtomicLevelAt(logLevel)
	logger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync()

	// Initialize database connection
	dbConfig := database.Config{
		Host:     globalConfig.Database.Host,
		Port:     globalConfig.Database.Port,
		User:     globalConfig.Database.User,
		Password: globalConfig.Database.Password,
		DBName:   globalConfig.Database.Name,
		SSLMode:  globalConfig.Database.SSLMode,
	}

	db, err := database.NewDB(dbConfig, logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize provider factory and create translator provider
	providerFactory := translator_provider.NewFactory(globalConfig)

	// You can change this to translator_provider.ProviderGemini to use Gemini instead
	provider, err := providerFactory.CreateProvider(translator_provider.ProviderGemini)
	if err != nil {
		logger.Fatal("failed to create translator provider", zap.Error(err))
	}

	// Initialize services
	translatorService := code_translator.NewCodeTranslatorService(logger, provider)

	svc := services.NewServices(translatorService)

	// Start the HTTP server
	runServer(logger, globalConfig, db, svc)
}

func runServer(logger *zap.Logger, cfg *types.Config, db *database.DB, svc *services.Services) {

	apiServer := api.NewGinServer(logger, svc)
	// Create HTTP server
	addr := cfg.Server.GetServerAddress()
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      apiServer.GetRouter(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting server", zap.String("address", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", zap.Error(err))
		}
	}()

	// Create channel to listen for interrupt signals (Ctrl+C)
	quit := make(chan os.Signal, 1)
	// Notify on SIGINT (Ctrl+C) and SIGTERM (kill command)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal
	<-quit
	logger.Info("shutting down server...")

	// Create a context with 10-second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}

package main

import (
	"fmt"
	"log"
	"rcode/platform/shutdown"
	"time"

	"rcode/config"
	"rcode/db"
	"rcode/web"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

func main() {
	// Initialize configuration
	config.Initialize()
	cfg := config.Get()

	// Log API endpoint configuration
	if cfg.AnthropicAPIURL != "https://api.anthropic.com/v1/messages" {
		logger.Info("Using proxy for Anthropic API", "url", cfg.AnthropicAPIURL)
	} else {
		logger.Info("Using direct connection to Anthropic API")
	}

	done := make(chan struct{}) // done channel will signal when shutdown complete
	shutdown.InitShutdownService(done)

	// Initialize database
	database, err := db.GetDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if database != nil {
			database.Close()
		}
	}()

	shutdown.RegisterHook(func(_ time.Duration) error {
		logger.F("Shutting down database")
		if err := database.Close(); err != nil {
			logger.LogErr(err, "Failed to close database")
		}
		logger.F("Database closed successfully")
		database = nil
		return nil
	})

	logger.Info("Database initialized successfully")

	go func() {
		s := rweb.NewServer(rweb.ServerOptions{
			Address: ":8000",
			Verbose: true,
		})

		// Add middleware for request logging
		s.Use(rweb.RequestInfo)
		s.ElementDebugRoutes()

		web.SetupRoutes(s)

		log.Printf("Starting RCode server on :8000")
		err = s.Run()
		if err != nil {
			logger.Err(err, "where", "at server exit")
		}
		logger.F("Server exited")
	}()

	// Block until done signal
	<-done
	fmt.Println("App exited")
}

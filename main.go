package main

import (
	"log"

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
	
	// Initialize database
	database, err := db.GetDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	logger.Info("Database initialized successfully")

	s := rweb.NewServer(rweb.ServerOptions{
		Address: ":8000",
		Verbose: true,
	})

	// Add middleware for request logging
	s.Use(rweb.RequestInfo)
	s.ElementDebugRoutes()

	web.SetupRoutes(s)

	log.Printf("Starting RCode server on :8000")
	log.Fatal(s.Run())
}

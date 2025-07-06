package main

import (
	"log"

	"rcode/db"
	"rcode/web"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

func main() {
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

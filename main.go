package main

import (
	"log"

	"rcode/handlers"

	"github.com/rohanthewiz/rweb"
)

func main() {
	// Create a new rweb server with options
	s := rweb.NewServer(rweb.ServerOptions{
		Address: ":8000",
		Verbose: true,
	})

	// Add middleware for request logging
	s.Use(rweb.RequestInfo)
	s.ElementDebugRoutes()

	// Setup routes
	handlers.SetupRoutes(s)

	// Start the server
	log.Printf("Starting RoCode server on :8000")
	log.Fatal(s.Run())
}

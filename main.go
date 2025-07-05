package main

import (
	"log"

	"github.com/rohanthewiz/rweb"
	"rcode/handlers"
)

func main() {
	// Create a new rweb server with options
	s := rweb.NewServer(rweb.ServerOptions{
		Address: ":8000",
		Verbose: true,
	})

	// Add middleware for request logging
	s.Use(rweb.RequestInfo)

	// Setup routes
	handlers.SetupRoutes(s)

	// Start the server
	log.Printf("Starting RoCode server on :8000")
	log.Fatal(s.Run())
}

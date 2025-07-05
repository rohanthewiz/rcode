package main

import (
	"log"

	"rcode/handlers"

	"github.com/rohanthewiz/rweb"
)

func main() {
	s := rweb.NewServer(rweb.ServerOptions{
		Address: ":8000",
		Verbose: true,
	})

	// Add middleware for request logging
	s.Use(rweb.RequestInfo)
	s.ElementDebugRoutes()

	handlers.SetupRoutes(s)

	log.Printf("Starting RoCode server on :8000")
	log.Fatal(s.Run())
}

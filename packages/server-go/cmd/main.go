package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rohanthewiz/logger"
	"github.com/sst/opencode/server-go/internal/server"
)

func main() {
	// Parse command line flags
	var (
		port    = flag.Int("port", 4096, "Server port")
		verbose = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()
	
	// Initialize logger
	if *verbose {
		logger.SetLevel(logger.LevelDebug)
	}
	
	// Create and start server
	srv := server.NewServer(*port)
	
	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		
		logger.Info("Shutting down server...")
		os.Exit(0)
	}()
	
	// Start the server
	logger.Info("Starting OpenCode Go server", "port", *port)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

// TestSSEHandler is a simple SSE endpoint for testing
func TestSSEHandler(c rweb.Context) error {
	logger.Info("Test SSE connection request received")

	// Set SSE headers
	c.Response().SetHeader("Content-Type", "text/event-stream")
	c.Response().SetHeader("Cache-Control", "no-cache")
	c.Response().SetHeader("Connection", "keep-alive")
	c.Response().SetHeader("Access-Control-Allow-Origin", "*")
	c.Response().SetHeader("X-Accel-Buffering", "no")

	// Write headers
	c.Response().WriteHeader(http.StatusOK)

	// Send initial message
	_, err := fmt.Fprintf(c.Response(), "event: connected\ndata: {\"message\": \"Test SSE connected\"}\n\n")
	if err != nil {
		logger.LogErr(err, "failed to send test SSE event")
		return err
	}

	// Flush immediately
	if flusher, ok := c.Response().(http.Flusher); ok {
		flusher.Flush()
		logger.Info("Test SSE initial event flushed")
	}

	// Send a few test messages
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)

		_, err := fmt.Fprintf(c.Response(), "data: {\"message\": \"Test message %d\"}\n\n", i)
		if err != nil {
			logger.LogErr(err, "failed to send test message")
			return err
		}

		if flusher, ok := c.Response().(http.Flusher); ok {
			flusher.Flush()
		}
	}

	// Send completion message
	_, _ = fmt.Fprintf(c.Response(), "event: complete\ndata: {\"message\": \"Test complete\"}\n\n")
	if flusher, ok := c.Response().(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

const (
	// Target Anthropic API URL
	targetURL = "https://api.anthropic.com/v1/messages"
)

func main() {
	s := rweb.NewServer(rweb.ServerOptions{
		Address: ":8001",
		Verbose: true,
	})

	// Proxy endpoint for Anthropic messages API
	s.Post("/v1/messages", proxyHandler)

	fmt.Println("Proxy server running on :8001")
	fmt.Println("Proxying requests to:", targetURL)
	log.Fatal(s.Run())
}

func proxyHandler(ctx rweb.Context) error {
	// Get the request body
	body := ctx.Request().Body()

	// Create new request to Anthropic
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return serr.Wrap(err, "failed to create proxy request")
	}

	// Copy all headers from original request
	for _, header := range ctx.Request().Headers() {
		req.Header.Add(header.Key, header.Value)
	}

	// Log the request for debugging
	fmt.Printf("Proxying request to %s\n", targetURL)
	fmt.Printf("Authorization: %s\n", req.Header.Get("Authorization"))
	fmt.Printf("anthropic-beta: %s\n", req.Header.Get("anthropic-beta"))
	fmt.Printf("Accept: %s\n", req.Header.Get("Accept"))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return serr.Wrap(err, "failed to send proxy request")
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			ctx.Response().SetHeader(key, value)
		}
	}

	// Set the status code
	ctx.Status(resp.StatusCode)

	// Check if this is a streaming response
	if strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
		// Handle SSE streaming
		fmt.Println("Streaming response detected")

		// Set up for streaming
		ctx.Response().SetHeader("Content-Type", "text/event-stream")
		ctx.Response().SetHeader("Cache-Control", "no-cache")
		ctx.Response().SetHeader("Connection", "keep-alive")

		// Stream the response by reading and writing chunks
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if err := ctx.Bytes(buf[:n]); err != nil {
					return serr.Wrap(err, "failed to write streaming response")
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return serr.Wrap(err, "failed to read streaming response")
			}
		}
	} else {
		// Regular response - copy body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return serr.Wrap(err, "failed to read response body")
		}

		if err := ctx.Bytes(body); err != nil {
			return serr.Wrap(err, "failed to write response")
		}
	}

	return nil
}

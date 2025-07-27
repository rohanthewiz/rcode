package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

const (
	// Target Anthropic API URL
	targetURL = "https://api.anthropic.com/v1/messages"
)

var (
	// Command line flags
	httpAddr  = flag.String("http", ":8001", "HTTP address to listen on")
	httpsAddr = flag.String("https", ":8443", "HTTPS address to listen on")
	certPath  = flag.String("cert", "", "Path to TLS certificate file (for Let's Encrypt: /etc/letsencrypt/live/yourdomain/fullchain.pem)")
	keyPath   = flag.String("key", "", "Path to TLS private key file (for Let's Encrypt: /etc/letsencrypt/live/yourdomain/privkey.pem)")
	tlsOnly   = flag.Bool("tls-only", false, "Only serve over HTTPS, redirect HTTP to HTTPS")
)

func main() {
	flag.Parse()

	// Check if we're running in HTTPS mode
	httpsEnabled := *certPath != "" && *keyPath != ""

	// Validate certificate files if HTTPS is enabled
	if httpsEnabled {
		if _, err := os.Stat(*certPath); os.IsNotExist(err) {
			log.Fatalf("Certificate file not found: %s", *certPath)
		}
		if _, err := os.Stat(*keyPath); os.IsNotExist(err) {
			log.Fatalf("Key file not found: %s", *keyPath)
		}
	}

	// Configure server options
	serverOpts := rweb.ServerOptions{
		Verbose: true,
	}

	if httpsEnabled {
		serverOpts.Address = *httpAddr // HTTP address for redirect server
		serverOpts.TLS = rweb.TLSCfg{
			UseTLS:   true,
			TLSAddr:  *httpsAddr,
			CertFile: *certPath,
			KeyFile:  *keyPath,
		}
	} else {
		serverOpts.Address = *httpAddr
	}

	s := rweb.NewServer(serverOpts)

	// Proxy endpoint for Anthropic messages API
	s.Post("/v1/messages", proxyHandler)

	// Start the server
	if httpsEnabled {
		fmt.Printf("HTTPS proxy server running on %s\n", *httpsAddr)
		fmt.Printf("Using certificate: %s\n", *certPath)
		fmt.Printf("Using private key: %s\n", *keyPath)
		
		if *tlsOnly {
			// Also start an HTTP server that redirects to HTTPS
			go func() {
				httpServer := rweb.NewServer(rweb.ServerOptions{
					Address: *httpAddr,
					Verbose: true,
				})
				httpServer.Use(httpsRedirectMiddleware(*httpsAddr))
				fmt.Printf("HTTP redirect server running on %s\n", *httpAddr)
				log.Fatal(httpServer.Run())
			}()
		}
		
		log.Fatal(s.Run())
	} else {
		fmt.Printf("HTTP proxy server running on %s\n", *httpAddr)
		fmt.Println("To enable HTTPS, provide -cert and -key flags with paths to your Let's Encrypt certificates")
		log.Fatal(s.Run())
	}
}

// httpsRedirectMiddleware redirects all HTTP requests to HTTPS
func httpsRedirectMiddleware(httpsAddr string) func(rweb.Context) error {
	return func(ctx rweb.Context) error {
		req := ctx.Request()
		
		// Get the host header
		host := ""
		for _, h := range req.Headers() {
			if strings.ToLower(h.Key) == "host" {
				host = h.Value
				break
			}
		}
		
		if host == "" {
			host = strings.Split(httpsAddr, ":")[0]
			if host == "" {
				host = "localhost"
			}
		}
		
		// Get the port from httpsAddr if it's not the default 443
		httpsPort := strings.Split(httpsAddr, ":")[1]
		if httpsPort != "443" {
			host = strings.Split(host, ":")[0] + ":" + httpsPort
		}
		
		// Get the request path - we'll use a simple approach
		// In a real scenario, you might need to check rweb's API documentation
		httpsURL := fmt.Sprintf("https://%s%s", host, "/v1/messages")
		ctx.Status(301)
		ctx.Response().SetHeader("Location", httpsURL)
		return nil
	}
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

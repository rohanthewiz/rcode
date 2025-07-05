package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/sst/opencode/server-go/internal/auth"
)

// Simple OAuth demo server
func main() {
	// Initialize auth storage
	anthropicAuth := auth.NewAnthropicAuth()

	// Auth start endpoint
	http.HandleFunc("/auth/anthropic/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		result, err := anthropicAuth.AuthorizeURL()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate auth URL: %v", err), http.StatusInternalServerError)
			return
		}

		// Open browser
		openBrowser(result.URL)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"url":      result.URL,
			"verifier": result.Verifier,
			"message":  "Please complete authentication in your browser",
		})
	})

	// Auth callback endpoint
	http.HandleFunc("/auth/anthropic/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Code     string `json:"code"`
			Verifier string `json:"verifier"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := anthropicAuth.Exchange(req.Code, req.Verifier); err != nil {
			http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Successfully authenticated with Anthropic Pro/Max",
		})
	})

	// Auth status endpoint
	http.HandleFunc("/auth/status", func(w http.ResponseWriter, r *http.Request) {
		storage := auth.NewStorage()
		creds, err := storage.Get("anthropic")
		
		status := map[string]interface{}{
			"anthropic": map[string]interface{}{
				"authenticated": false,
			},
		}

		if err == nil && creds != nil {
			status["anthropic"] = map[string]interface{}{
				"authenticated": true,
				"type":          creds.Type,
				"hasToken":      creds.Access != "",
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// Test access token endpoint
	http.HandleFunc("/auth/test", func(w http.ResponseWriter, r *http.Request) {
		token, err := anthropicAuth.Access()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get access token: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hasToken": token != "",
			"message":  "Token retrieved successfully",
		})
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Println("OAuth Demo Server running on :4096")
	log.Println("Endpoints:")
	log.Println("  POST /auth/anthropic/start    - Start OAuth flow")
	log.Println("  POST /auth/anthropic/callback - Complete OAuth")
	log.Println("  GET  /auth/status            - Check auth status")
	log.Println("  GET  /auth/test              - Test token access")
	log.Println("  GET  /health                 - Health check")
	
	log.Fatal(http.ListenAndServe(":4096", nil))
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}
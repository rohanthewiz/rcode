package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Test basic auth endpoint structure
	http.HandleFunc("/auth/anthropic/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		fmt.Fprintf(w, `{"message": "Auth endpoint is working", "url": "https://claude.ai/oauth/authorize"}`)
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "ok"}`)
	})
	
	log.Println("Test server running on :4096")
	log.Println("Try: curl -X POST http://localhost:4096/auth/anthropic/start")
	log.Fatal(http.ListenAndServe(":4096", nil))
}
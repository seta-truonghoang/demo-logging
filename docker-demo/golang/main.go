package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type InfoResponse struct {
	Language  string    `json:"language"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Hostname  string    `json:"hostname"`
	Version   string    `json:"version"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Log the incoming request
		log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		response := InfoResponse{
			Language:  "Go",
			Message:   "Xin chào từ Docker Container tối ưu cho Go!",
			Timestamp: time.Now(),
			Hostname:  hostname,
			Version:   "1.0.0",
		}
		json.NewEncoder(w).Encode(response)
	})

	log.Printf("Go application is starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

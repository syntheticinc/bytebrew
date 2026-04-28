package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "/tmp/mcp-requests.jsonl"
	}

	mux := http.NewServeMux()
	h := NewMCPHandler(logFile)
	mux.HandleFunc("/mcp", h.Handle)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9556"
	}

	log.Printf("Mock MCP HTTP server starting on :%s (log: %s)", port, logFile)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

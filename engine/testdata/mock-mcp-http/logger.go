package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// requestLogEntry represents a single logged request.
type requestLogEntry struct {
	Timestamp string              `json:"timestamp"`
	Headers   map[string][]string `json:"headers"`
	Body      json.RawMessage     `json:"body"`
}

// logRequest appends a JSONL entry with the request headers and body.
func logRequest(logFile string, headers http.Header, body []byte) error {
	entry := requestLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Headers:   headers,
		Body:      body,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}
	line = append(line, '\n')

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("write log entry: %w", err)
	}

	return nil
}

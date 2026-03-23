package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// JSON-RPC types
// ---------------------------------------------------------------------------

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// MCP tool definitions
// ---------------------------------------------------------------------------

var tools = []map[string]interface{}{
	{
		"name":        "search_docs",
		"description": "Search ByteBrew Engine documentation. Returns relevant passages from the official docs. Use this to answer questions about configuration, API, deployment, agents, tools, MCP servers, triggers, knowledge/RAG, and examples.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Natural language search query about ByteBrew Engine",
				},
			},
			"required": []string{"query"},
		},
	},
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type Config struct {
	Port       string
	EngineURL  string
	EngineUser string
	EnginePass string
}

func loadConfig() Config {
	return Config{
		Port:       envOr("PORT", "8090"),
		EngineURL:  envOr("ENGINE_URL", "http://localhost:8443"),
		EngineUser: envOr("ENGINE_USER", "admin"),
		EnginePass: envOr("ENGINE_PASS", "changeme"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// SSE session
// ---------------------------------------------------------------------------

type sseSession struct {
	id       string
	messages chan []byte
	done     chan struct{}
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

type Server struct {
	cfg         Config
	engineToken string
	tokenMu     sync.Mutex
	mu          sync.Mutex
	sessions    map[string]*sseSession
}

func NewServer(cfg Config) *Server {
	return &Server{
		cfg:      cfg,
		sessions: make(map[string]*sseSession),
	}
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sess := &sseSession{
		id:       uuid.New().String(),
		messages: make(chan []byte, 64),
		done:     make(chan struct{}),
	}

	s.mu.Lock()
	s.sessions[sess.id] = sess
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sess.id)
		s.mu.Unlock()
		close(sess.done)
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send endpoint event.
	fmt.Fprintf(w, "event: endpoint\ndata: /messages?sessionId=%s\n\n", sess.id)
	flusher.Flush()

	log.Printf("[sse] session %s connected", sess.id)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[sse] session %s disconnected", sess.id)
			return
		case msg, ok := <-sess.messages:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	log.Printf("[rpc] session=%s method=%s id=%v", sessionID, req.Method, req.ID)

	// Notifications have no ID and don't expect a response.
	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := s.handleRPC(r.Context(), &req)

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[rpc] marshal error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	select {
	case sess.messages <- data:
	default:
		log.Printf("[rpc] session %s buffer full, dropping response", sessionID)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// ---------------------------------------------------------------------------
// JSON-RPC dispatch
// ---------------------------------------------------------------------------

func (s *Server) handleRPC(ctx context.Context, req *jsonRPCRequest) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)},
		}
	}
}

func (s *Server) handleInitialize(req *jsonRPCRequest) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "bytebrew-docs",
				"version": "1.0.0",
			},
		},
	}
}

func (s *Server) handleToolsList(req *jsonRPCRequest) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *jsonRPCRequest) *jsonRPCResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: "invalid params"},
		}
	}

	if params.Name != "search_docs" {
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", params.Name)},
		}
	}

	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: "invalid arguments"},
		}
	}

	if args.Query == "" {
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: "query is required"},
		}
	}

	result, err := s.queryEngine(ctx, args.Query)
	if err != nil {
		log.Printf("[engine] query error: %v", err)
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Error querying docs: %v", err)},
				},
			},
		}
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": result},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Engine integration
// ---------------------------------------------------------------------------

func (s *Server) login(ctx context.Context) (string, error) {
	body, err := json.Marshal(map[string]string{
		"username": s.cfg.EngineUser,
		"password": s.cfg.EnginePass,
	})
	if err != nil {
		return "", fmt.Errorf("marshal login: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.EngineURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed (%d): %s", resp.StatusCode, string(b))
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	return loginResp.Token, nil
}

func (s *Server) getToken(ctx context.Context) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	if s.engineToken != "" {
		return s.engineToken, nil
	}

	token, err := s.login(ctx)
	if err != nil {
		return "", err
	}
	s.engineToken = token
	return token, nil
}

func (s *Server) queryEngine(ctx context.Context, query string) (string, error) {
	token, err := s.getToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	result, err := s.doChat(ctx, token, query)
	if err != nil {
		// Token may have expired — retry with fresh login.
		s.tokenMu.Lock()
		s.engineToken = ""
		s.tokenMu.Unlock()

		token, loginErr := s.getToken(ctx)
		if loginErr != nil {
			return "", fmt.Errorf("re-login: %w", loginErr)
		}
		result, err = s.doChat(ctx, token, query)
		if err != nil {
			return "", fmt.Errorf("chat after re-login: %w", err)
		}
	}
	return result, nil
}

func (s *Server) doChat(ctx context.Context, token, query string) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"message": query,
		"stream":  true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal chat: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.EngineURL+"/api/v1/agents/docs-assistant/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chat failed (%d): %s", resp.StatusCode, string(b))
	}

	return s.parseSSEResponse(resp.Body)
}

func (s *Server) parseSSEResponse(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	var (
		content   strings.Builder
		eventType string
	)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			switch eventType {
			case "message_delta", "message":
				var payload struct {
					Content string `json:"content"`
				}
				if err := json.Unmarshal([]byte(data), &payload); err == nil && payload.Content != "" {
					content.WriteString(payload.Content)
				}
			case "done":
				// Stream finished.
			}
			eventType = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return content.String(), fmt.Errorf("scan sse: %w", err)
	}

	if content.Len() == 0 {
		return "(no response from docs-assistant)", nil
	}
	return content.String(), nil
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	cfg := loadConfig()
	srv := NewServer(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", srv.handleSSE)
	mux.HandleFunc("/messages", srv.handleMessages)
	mux.HandleFunc("/health", srv.handleHealth)

	httpSrv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // SSE needs no write timeout
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("[server] listening on :%s", cfg.Port)
		log.Printf("[server] engine: %s, agent: docs-assistant", cfg.EngineURL)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[server] shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[server] shutdown error: %v", err)
	}
	log.Println("[server] stopped")
}

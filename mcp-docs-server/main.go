package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
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
// MCP Tool definitions
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
	Port        string
	DatabaseURL string
	OllamaURL   string
	EmbedModel  string
	AgentName   string
}

func loadConfig() Config {
	return Config{
		Port:        envOr("PORT", "8090"),
		DatabaseURL: envOr("DATABASE_URL", "postgres://bytebrew:bytebrew@localhost:5432/bytebrew?sslmode=disable"),
		OllamaURL:   envOr("OLLAMA_URL", "http://localhost:11434"),
		EmbedModel:  envOr("EMBED_MODEL", "nomic-embed-text"),
		AgentName:   envOr("AGENT_NAME", "docs-assistant"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// SSE Session
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
	cfg      Config
	db       *sql.DB
	mu       sync.Mutex
	sessions map[string]*sseSession
}

func NewServer(cfg Config) (*Server, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Server{
		cfg:      cfg,
		db:       db,
		sessions: make(map[string]*sseSession),
	}, nil
}

// ---------------------------------------------------------------------------
// HTTP Handlers
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

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
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(w, "event: endpoint\ndata: /messages?sessionId=%s\n\n", sess.id)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
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
	s.mu.Lock()
	sess, exists := s.sessions[sessionID]
	s.mu.Unlock()

	if !exists {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Notifications (no ID) — no response needed
	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var resp jsonRPCResponse
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":   map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":     map[string]interface{}{"name": "bytebrew-docs", "version": "1.0.0"},
		}
	case "tools/list":
		resp.Result = map[string]interface{}{"tools": tools}
	case "tools/call":
		result := s.handleToolCall(r.Context(), req.Params)
		resp.Result = result
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}

	data, _ := json.Marshal(resp)
	select {
	case sess.messages <- data:
	default:
		log.Printf("session %s: message channel full, dropping", sessionID)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleToolCall(ctx context.Context, params json.RawMessage) interface{} {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("error: %v", err)},
			},
		}
	}

	if p.Name != "search_docs" {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "unknown tool: " + p.Name},
			},
		}
	}

	var args struct {
		Query string `json:"query"`
	}
	json.Unmarshal(p.Arguments, &args)

	result, err := s.searchDocs(ctx, args.Query)
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("search error: %v", err)},
			},
		}
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": result},
		},
	}
}

// ---------------------------------------------------------------------------
// Direct RAG search (no LLM — embedding + vector + keyword)
// ---------------------------------------------------------------------------

func (s *Server) searchDocs(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "Please provide a search query.", nil
	}

	// 1. Get embedding from Ollama
	embedding, err := s.embed(ctx, query)
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}

	// 2. Vector search
	vectorResults, err := s.vectorSearch(ctx, embedding, 10)
	if err != nil {
		return "", fmt.Errorf("vector search: %w", err)
	}

	// 3. Keyword search (hybrid)
	words := strings.Fields(query)
	var keywordResults []searchResult
	for _, word := range words {
		if len(word) < 4 {
			continue
		}
		kw, err := s.keywordSearch(ctx, word, 3)
		if err != nil {
			continue
		}
		keywordResults = append(keywordResults, kw...)
	}

	// 4. Merge and deduplicate
	seen := make(map[string]bool)
	var merged []searchResult
	for _, r := range vectorResults {
		if !seen[r.id] {
			seen[r.id] = true
			merged = append(merged, r)
		}
	}
	for _, r := range keywordResults {
		if !seen[r.id] {
			seen[r.id] = true
			merged = append(merged, r)
		}
	}

	if len(merged) == 0 {
		return "No results found in the documentation for: \"" + query + "\". Try different search terms.", nil
	}

	// 5. Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Documentation search results for \"%s\"\n\n", query))
	limit := 10
	if len(merged) < limit {
		limit = len(merged)
	}
	for i := 0; i < limit; i++ {
		r := merged[i]
		sb.WriteString(fmt.Sprintf("### Result %d (Source: %s)\n", i+1, r.source))
		sb.WriteString(r.content)
		sb.WriteString("\n\n")
	}
	return sb.String(), nil
}

type searchResult struct {
	id      string
	content string
	source  string
}

func (s *Server) embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(map[string]interface{}{
		"model": s.cfg.EmbedModel,
		"input": text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.OllamaURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned %d", resp.StatusCode)
	}

	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}
	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	return result.Embeddings[0], nil
}

func (s *Server) vectorSearch(ctx context.Context, embedding []float32, limit int) ([]searchResult, error) {
	// Format embedding as pgvector literal: [0.1,0.2,...]
	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = fmt.Sprintf("%f", v)
	}
	vecStr := "[" + strings.Join(parts, ",") + "]"

	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.content, COALESCE(d.file_name, c.agent_name) as source
		 FROM knowledge_chunks c
		 LEFT JOIN knowledge_documents d ON c.document_id = d.id
		 WHERE c.agent_name = $1
		 ORDER BY c.embedding <=> $2::vector
		 LIMIT $3`,
		s.cfg.AgentName, vecStr, limit)
	if err != nil {
		return nil, fmt.Errorf("vector query: %w", err)
	}
	defer rows.Close()

	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.id, &r.content, &r.source); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

func (s *Server) keywordSearch(ctx context.Context, keyword string, limit int) ([]searchResult, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.content, COALESCE(d.file_name, c.agent_name) as source
		 FROM knowledge_chunks c
		 LEFT JOIN knowledge_documents d ON c.document_id = d.id
		 WHERE c.agent_name = $1 AND c.content ILIKE '%' || $2 || '%'
		 LIMIT $3`,
		s.cfg.AgentName, keyword, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword query: %w", err)
	}
	defer rows.Close()

	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.id, &r.content, &r.source); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	cfg := loadConfig()

	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}
	defer srv.db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/sse", srv.handleSSE)
	mux.HandleFunc("/messages", srv.handleMessages)

	httpSrv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		httpSrv.Shutdown(shutCtx)
	}()

	log.Printf("MCP docs server starting on :%s (direct RAG, no LLM)", cfg.Port)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

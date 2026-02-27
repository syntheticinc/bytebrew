package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// WebSearchResult represents a single search result
type WebSearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// WebSearchOptions holds optional parameters for search
type WebSearchOptions struct {
	MaxResults     int
	IncludeDomains []string
	ExcludeDomains []string
}

// WebFetchResult represents fetched page content
type WebFetchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// TavilyProvider implements web search and fetch via Tavily API
type TavilyProvider struct {
	apiKey string
	client *http.Client
}

// NewTavilyProvider creates a new TavilyProvider
func NewTavilyProvider(apiKey string) *TavilyProvider {
	return &TavilyProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// tavilySearchRequest is the request body for Tavily search API
type tavilySearchRequest struct {
	Query          string   `json:"query"`
	MaxResults     int      `json:"max_results,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	APIKey         string   `json:"api_key"`
}

// tavilySearchResponse is the response from Tavily search API
type tavilySearchResponse struct {
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

// Search executes a web search query via Tavily API
func (p *TavilyProvider) Search(ctx context.Context, query string, opts WebSearchOptions) ([]WebSearchResult, error) {
	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}

	reqBody := tavilySearchRequest{
		Query:          query,
		MaxResults:     maxResults,
		IncludeDomains: opts.IncludeDomains,
		ExcludeDomains: opts.ExcludeDomains,
		APIKey:         p.apiKey,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("tavily search marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tavily search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily search: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tavily search read body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		slog.WarnContext(ctx, "[TavilyProvider] rate limited", "status", resp.StatusCode)
		return nil, fmt.Errorf("tavily API rate limit exceeded, please try again later")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily search: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var tavilyResp tavilySearchResponse
	if err := json.Unmarshal(respBody, &tavilyResp); err != nil {
		return nil, fmt.Errorf("tavily search unmarshal: %w", err)
	}

	results := make([]WebSearchResult, 0, len(tavilyResp.Results))
	for _, r := range tavilyResp.Results {
		results = append(results, WebSearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Score:   r.Score,
		})
	}

	slog.InfoContext(ctx, "[TavilyProvider] search completed", "query", query, "results", len(results))
	return results, nil
}

// tavilyExtractRequest is the request body for Tavily extract API
type tavilyExtractRequest struct {
	URLs   []string `json:"urls"`
	APIKey string   `json:"api_key"`
}

// tavilyExtractResponse is the response from Tavily extract API
type tavilyExtractResponse struct {
	Results []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Content string `json:"raw_content"`
	} `json:"results"`
}

// Fetch extracts content from a URL via Tavily extract API
func (p *TavilyProvider) Fetch(ctx context.Context, url string) (*WebFetchResult, error) {
	reqBody := tavilyExtractRequest{
		URLs:   []string{url},
		APIKey: p.apiKey,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("tavily extract marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/extract", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tavily extract request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily extract: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tavily extract read body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		slog.WarnContext(ctx, "[TavilyProvider] rate limited", "status", resp.StatusCode)
		return nil, fmt.Errorf("tavily API rate limit exceeded, please try again later")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily extract: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var tavilyResp tavilyExtractResponse
	if err := json.Unmarshal(respBody, &tavilyResp); err != nil {
		return nil, fmt.Errorf("tavily extract unmarshal: %w", err)
	}

	if len(tavilyResp.Results) == 0 {
		return nil, fmt.Errorf("tavily extract: no content extracted from %s", url)
	}

	r := tavilyResp.Results[0]
	slog.InfoContext(ctx, "[TavilyProvider] fetch completed", "url", url, "content_length", len(r.Content))

	return &WebFetchResult{
		URL:     r.URL,
		Title:   r.Title,
		Content: r.Content,
	}, nil
}

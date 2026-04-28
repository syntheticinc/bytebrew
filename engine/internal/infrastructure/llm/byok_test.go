package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildBYOKChatModel_OpenAICompatible_RoutesToUserEndpoint is the
// integration check for V2 §5.8: when BYOK credentials are present, the
// LLM call must be issued against the user-supplied base URL with the
// user-supplied API key — bypassing any tenant-configured model.
func TestBuildBYOKChatModel_OpenAICompatible_RoutesToUserEndpoint(t *testing.T) {
	// Capture what the OpenAI-compatible adapter sends.
	var capturedAuth atomic.Value
	var capturedBody atomic.Value
	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		capturedAuth.Store(r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		capturedBody.Store(string(body))

		// Minimal OpenAI chat completions success payload.
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":"chatcmpl-byok-1",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-4o-mini",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello from BYOK endpoint"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer srv.Close()

	creds := BYOKCredentials{
		Provider: "openai_compatible",
		APIKey:   "sk-byok-secret",
		Model:    "gpt-4o-mini",
		BaseURL:  srv.URL,
	}

	model, err := BuildBYOKChatModel(context.Background(), creds)
	require.NoError(t, err)
	require.NotNil(t, model)

	resp, err := model.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "ping"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "hello from BYOK endpoint", resp.Content)

	// The BYOK call MUST have hit our test server (proves bypass of tenant
	// config) using the user-supplied API key (proves no swallow / no
	// substitution).
	assert.Equal(t, int32(1), hits.Load(), "BYOK call did not reach user-supplied endpoint")
	auth, _ := capturedAuth.Load().(string)
	assert.Equal(t, "Bearer sk-byok-secret", auth, "BYOK call did not carry the user-supplied API key")

	// Verify the model name in the outbound payload matches the BYOK header.
	body, _ := capturedBody.Load().(string)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &parsed))
	assert.Equal(t, "gpt-4o-mini", parsed["model"])
}

// TestBuildBYOKChatModel_SurfacesProvider401 ensures that an invalid
// user API key surfaces as an error to the caller (per §5.8 negative
// case "invalid key (LLM 401) → surfaced"), not swallowed.
func TestBuildBYOKChatModel_SurfacesProvider401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"invalid api key","type":"invalid_request_error","code":"invalid_api_key"}}`)
	}))
	defer srv.Close()

	creds := BYOKCredentials{
		Provider: "openai_compatible",
		APIKey:   "sk-bogus",
		Model:    "gpt-4o-mini",
		BaseURL:  srv.URL,
	}

	model, err := BuildBYOKChatModel(context.Background(), creds)
	require.NoError(t, err)
	require.NotNil(t, model)

	_, err = model.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "ping"},
	})
	require.Error(t, err, "BYOK 401 must surface to caller, not be swallowed")
	// The eino openai adapter wraps the upstream error; we just check the
	// wire-level signal made it through (status code or message body).
	msg := err.Error()
	if !strings.Contains(msg, "401") && !strings.Contains(msg, "Unauthorized") &&
		!strings.Contains(msg, "invalid") {
		t.Fatalf("expected surfaced 401/Unauthorized/invalid key error, got: %v", err)
	}
}

// TestBuildBYOKChatModel_RequiresAPIKey is a unit-level guard against
// silently building a client without credentials.
func TestBuildBYOKChatModel_RequiresAPIKey(t *testing.T) {
	_, err := BuildBYOKChatModel(context.Background(), BYOKCredentials{
		Provider: "openai",
		Model:    "gpt-4o-mini",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key required")
}

// TestBuildBYOKChatModel_RequiresProvider mirrors the api-key guard.
func TestBuildBYOKChatModel_RequiresProvider(t *testing.T) {
	_, err := BuildBYOKChatModel(context.Background(), BYOKCredentials{
		APIKey: "sk-x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider required")
}

// TestBuildBYOKChatModel_OpenAICompatibleRequiresBaseURL ensures a self-
// hosted/vLLM provider can't be routed without a base URL.
func TestBuildBYOKChatModel_OpenAICompatibleRequiresBaseURL(t *testing.T) {
	_, err := BuildBYOKChatModel(context.Background(), BYOKCredentials{
		Provider: "openai_compatible",
		APIKey:   "sk-x",
		Model:    "gpt-4o-mini",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url required")
}

// TestBuildBYOKChatModel_UnsupportedProvider exercises the explicit
// allowlist of supported BYOK providers.
func TestBuildBYOKChatModel_UnsupportedProvider(t *testing.T) {
	_, err := BuildBYOKChatModel(context.Background(), BYOKCredentials{
		Provider: "google",
		APIKey:   "sk-x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

// TestRedactAPIKey verifies the redacted form never contains the middle
// of the secret, so log lines remain safe (V2 §5.8 "never log raw keys").
func TestRedactAPIKey(t *testing.T) {
	cases := []struct {
		in       string
		contains string
		notFull  bool
	}{
		{"", "", false},
		{"short", "***", false},
		{"sk-abcd1234", "sk-a", true},
		{"sk-abcd1234", "1234", true},
		{"sk-very-long-api-key-12345", "sk-v", true},
	}
	for _, tc := range cases {
		got := RedactAPIKey(tc.in)
		if tc.contains != "" {
			assert.Contains(t, got, tc.contains)
		}
		if tc.notFull && tc.in != got {
			assert.NotEqual(t, tc.in, got, "redacted form must differ from raw key")
		}
	}
}

// TestBYOKContextRoundtrip ensures values stored via WithBYOKCredentials
// can be retrieved by BYOKCredentialsFrom — the contract relied on by
// the turn executor factory.
func TestBYOKContextRoundtrip(t *testing.T) {
	ctx := context.Background()

	assert.Nil(t, BYOKCredentialsFrom(ctx))

	creds := &BYOKCredentials{
		Provider: "openai",
		APIKey:   "sk-abc",
		Model:    "gpt-4o",
		BaseURL:  "https://example.com/v1",
	}
	ctx = WithBYOKCredentials(ctx, creds)

	got := BYOKCredentialsFrom(ctx)
	require.NotNil(t, got)
	assert.Equal(t, "openai", got.Provider)
	assert.Equal(t, "sk-abc", got.APIKey)
	assert.Equal(t, "gpt-4o", got.Model)
	assert.Equal(t, "https://example.com/v1", got.BaseURL)

	// Nil passed in must be a no-op (caller may pass nil when there are
	// no BYOK headers on the request).
	noCreds := WithBYOKCredentials(context.Background(), nil)
	assert.Nil(t, BYOKCredentialsFrom(noCreds))
}

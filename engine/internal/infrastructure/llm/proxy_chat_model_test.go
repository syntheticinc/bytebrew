package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- Generate ----------

func TestProxyChatModel_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format.
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/proxy/llm", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var req openAIRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "supervisor", req.Role)
		assert.False(t, req.Stream)
		require.Len(t, req.Messages, 2)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "You are helpful", req.Messages[0].Content)
		assert.Equal(t, "user", req.Messages[1].Role)
		assert.Equal(t, "Hello", req.Messages[1].Content)

		resp := openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "Hi there!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "test-token", "supervisor")

	input := []*schema.Message{
		{Role: schema.System, Content: "You are helpful"},
		{Role: schema.User, Content: "Hello"},
	}

	msg, err := proxy.Generate(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, schema.Assistant, msg.Role)
	assert.Equal(t, "Hi there!", msg.Content)
}

func TestProxyChatModel_Generate_WithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{
					Role:    "assistant",
					Content: "",
					ToolCalls: []openAIToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: openAIFunctionCall{
								Name:      "read_file",
								Arguments: `{"path":"main.go"}`,
							},
						},
					},
				}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "coder")
	msg, err := proxy.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "Read main.go"},
	})

	require.NoError(t, err)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "call_123", msg.ToolCalls[0].ID)
	assert.Equal(t, "read_file", msg.ToolCalls[0].Function.Name)
	assert.Equal(t, `{"path":"main.go"}`, msg.ToolCalls[0].Function.Arguments)
}

func TestProxyChatModel_Generate_ToolMessageRoundTrip(t *testing.T) {
	var capturedReq openAIRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)

		resp := openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "Done"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "coder")

	input := []*schema.Message{
		{Role: schema.User, Content: "Read file"},
		{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"x.go"}`,
					},
				},
			},
		},
		{
			Role:       schema.Tool,
			ToolCallID: "call_1",
			Content:    "file contents here",
		},
	}

	_, err := proxy.Generate(context.Background(), input)
	require.NoError(t, err)

	// Verify tool call was serialized.
	require.Len(t, capturedReq.Messages, 3)
	assert.Len(t, capturedReq.Messages[1].ToolCalls, 1)
	assert.Equal(t, "call_1", capturedReq.Messages[1].ToolCalls[0].ID)

	// Verify tool result message.
	assert.Equal(t, "tool", capturedReq.Messages[2].Role)
	assert.Equal(t, "call_1", capturedReq.Messages[2].ToolCallID)
	assert.Equal(t, "file contents here", capturedReq.Messages[2].Content)
}

// ---------- Errors ----------

func TestProxyChatModel_Generate_402_QuotaExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte("quota exhausted"))
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "supervisor")
	_, err := proxy.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQuotaExhausted)
	assert.Contains(t, err.Error(), "quota exhausted")
}

func TestProxyChatModel_Generate_429_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "supervisor")
	_, err := proxy.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestProxyChatModel_Generate_500_GenericError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "supervisor")
	_, err := proxy.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrQuotaExhausted)
	assert.NotErrorIs(t, err, ErrRateLimited)
	assert.Contains(t, err.Error(), "HTTP 500")
}

// ---------- WithTools ----------

func TestProxyChatModel_WithTools_ToolsSentInRequest(t *testing.T) {
	var capturedReq openAIRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)

		resp := openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "coder")

	tools := []*schema.ToolInfo{
		{
			Name: "read_file",
			Desc: "Read a file from disk",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"path": {Type: "string", Desc: "file path", Required: true},
			}),
		},
	}

	withTools, err := proxy.WithTools(tools)
	require.NoError(t, err)

	_, err = withTools.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "read main.go"},
	})
	require.NoError(t, err)

	require.Len(t, capturedReq.Tools, 1)
	assert.Equal(t, "function", capturedReq.Tools[0].Type)
	assert.Equal(t, "read_file", capturedReq.Tools[0].Function.Name)
	assert.Equal(t, "Read a file from disk", capturedReq.Tools[0].Function.Description)
	assert.NotNil(t, capturedReq.Tools[0].Function.Parameters)
}

func TestProxyChatModel_WithTools_DoesNotMutateOriginal(t *testing.T) {
	proxy := NewProxyChatModel("http://localhost", "token", "coder")

	tools := []*schema.ToolInfo{
		{Name: "tool1", Desc: "desc"},
	}

	withTools, err := proxy.WithTools(tools)
	require.NoError(t, err)

	// Original should have no tools.
	assert.Empty(t, proxy.tools)

	// New model should have tools.
	proxyWithTools := withTools.(*ProxyChatModel)
	assert.Len(t, proxyWithTools.tools, 1)
}

// ---------- Stream ----------

func TestProxyChatModel_Stream_SSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		chunks := []string{
			`{"choices":[{"delta":{"role":"assistant","content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" world"}}]}`,
			`{"choices":[{"delta":{"content":"!"}}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}

		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "supervisor")

	reader, err := proxy.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	require.NoError(t, err)

	var contents []string
	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		contents = append(contents, msg.Content)
	}

	assert.Equal(t, []string{"Hello", " world", "!"}, contents)
}

func TestProxyChatModel_Stream_SSE_WithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []string{
			`{"choices":[{"delta":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"id":"","type":"","function":{"name":"","arguments":"\"main.go\"}"}}]}}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "coder")

	reader, err := proxy.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "read file"},
	})
	require.NoError(t, err)

	var messages []*schema.Message
	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		messages = append(messages, msg)
	}

	// First chunk should have tool call with partial args.
	require.GreaterOrEqual(t, len(messages), 1)
	require.Len(t, messages[0].ToolCalls, 1)
	assert.Equal(t, "call_1", messages[0].ToolCalls[0].ID)
	assert.Equal(t, "read_file", messages[0].ToolCalls[0].Function.Name)
}

func TestProxyChatModel_Stream_402_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte("no quota"))
	}))
	defer server.Close()

	proxy := NewProxyChatModel(server.URL, "token", "supervisor")

	_, err := proxy.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQuotaExhausted)
}

// ---------- conversion helpers ----------

func TestSchemaMessageToOpenAI_AllFields(t *testing.T) {
	msg := &schema.Message{
		Role:       schema.Assistant,
		Content:    "hello",
		Name:       "agent",
		ToolCallID: "tc-1",
		ToolCalls: []schema.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: schema.FunctionCall{
					Name:      "search",
					Arguments: `{"q":"test"}`,
				},
			},
		},
	}

	oai := schemaMessageToOpenAI(msg)

	assert.Equal(t, "assistant", oai.Role)
	assert.Equal(t, "hello", oai.Content)
	assert.Equal(t, "agent", oai.Name)
	assert.Equal(t, "tc-1", oai.ToolCallID)
	require.Len(t, oai.ToolCalls, 1)
	assert.Equal(t, "call-1", oai.ToolCalls[0].ID)
	assert.Equal(t, "function", oai.ToolCalls[0].Type)
	assert.Equal(t, "search", oai.ToolCalls[0].Function.Name)
}

func TestOaiMessageToSchema_AllFields(t *testing.T) {
	oai := &openAIMessage{
		Role:       "assistant",
		Content:    "result",
		Name:       "bot",
		ToolCallID: "tc-2",
		ToolCalls: []openAIToolCall{
			{
				ID:   "c2",
				Type: "function",
				Function: openAIFunctionCall{
					Name:      "write",
					Arguments: `{"file":"x"}`,
				},
			},
		},
	}

	msg := oaiMessageToSchema(oai)

	assert.Equal(t, schema.Assistant, msg.Role)
	assert.Equal(t, "result", msg.Content)
	assert.Equal(t, "bot", msg.Name)
	assert.Equal(t, "tc-2", msg.ToolCallID)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "c2", msg.ToolCalls[0].ID)
	assert.Equal(t, "write", msg.ToolCalls[0].Function.Name)
}

// ---------- parseSSEStream ----------

func TestParseSSEStream_Done(t *testing.T) {
	sse := strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n")

	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer sw.Close()
		err := parseSSEStream(sse, sw)
		assert.NoError(t, err)
	}()

	msg, err := sr.Recv()
	require.NoError(t, err)
	assert.Equal(t, "hi", msg.Content)

	_, err = sr.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestParseSSEStream_SkipsMalformedChunks(t *testing.T) {
	sse := strings.NewReader("data: {invalid-json}\ndata: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")

	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer sw.Close()
		parseSSEStream(sse, sw)
	}()

	msg, err := sr.Recv()
	require.NoError(t, err)
	assert.Equal(t, "ok", msg.Content)

	_, err = sr.Recv()
	assert.Equal(t, io.EOF, err)
}

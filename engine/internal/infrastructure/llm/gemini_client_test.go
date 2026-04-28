package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiChatModel_URLConstruction(t *testing.T) {
	client := NewGeminiChatModel("test-key", "gemini-3.1-pro")

	assert.Equal(t,
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro:generateContent",
		client.generateContentURL(),
	)
	assert.Equal(t,
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro:streamGenerateContent?alt=sse",
		client.streamContentURL(),
	)
}

func TestGeminiChatModel_CustomBaseURL(t *testing.T) {
	client := NewGeminiChatModel("test-key", "gemini-3.1-pro",
		WithGeminiBaseURL("https://custom.api.example.com/v1beta"),
	)

	assert.Equal(t,
		"https://custom.api.example.com/v1beta/models/gemini-3.1-pro:generateContent",
		client.generateContentURL(),
	)
}

func TestGeminiChatModel_AuthHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-api-key", r.Header.Get("x-goog-api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "Hello!"}},
					},
					FinishReason: "STOP",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiChatModel("test-api-key", "gemini-3.1-pro",
		WithGeminiBaseURL(ts.URL),
	)

	msg, err := client.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "Hello"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", msg.Content)
}

func TestGeminiChatModel_Generate_TextResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body
		var req geminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Contents, 1)
		assert.Equal(t, "user", req.Contents[0].Role)
		assert.Equal(t, "What is 2+2?", req.Contents[0].Parts[0].Text)

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "4"}},
					},
					FinishReason: "STOP",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiChatModel("key", "gemini-3.1-pro", WithGeminiBaseURL(ts.URL))

	msg, err := client.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "What is 2+2?"},
	})

	require.NoError(t, err)
	assert.Equal(t, schema.Assistant, msg.Role)
	assert.Equal(t, "4", msg.Content)
}

func TestGeminiChatModel_Generate_ToolCallResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role: "model",
						Parts: []geminiPart{
							{
								FunctionCall: &geminiFunctionCall{
									Name: "get_weather",
									Args: map[string]interface{}{
										"location": "London",
									},
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiChatModel("key", "gemini-3.1-pro", WithGeminiBaseURL(ts.URL))

	msg, err := client.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "Weather in London?"},
	})

	require.NoError(t, err)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "get_weather", msg.ToolCalls[0].Function.Name)
	assert.JSONEq(t, `{"location":"London"}`, msg.ToolCalls[0].Function.Arguments)
}

func TestGeminiChatModel_Generate_SystemInstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		// System message should be in systemInstruction, not in contents
		require.NotNil(t, req.SystemInstruction)
		assert.Equal(t, "Be helpful.", req.SystemInstruction.Parts[0].Text)

		// Only user message in contents
		require.Len(t, req.Contents, 1)
		assert.Equal(t, "user", req.Contents[0].Role)

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "OK"}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiChatModel("key", "gemini-3.1-pro", WithGeminiBaseURL(ts.URL))

	msg, err := client.Generate(context.Background(), []*schema.Message{
		{Role: schema.System, Content: "Be helpful."},
		{Role: schema.User, Content: "Hello"},
	})

	require.NoError(t, err)
	assert.Equal(t, "OK", msg.Content)
}

func TestGeminiChatModel_Generate_WithTools(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		// Verify tools are sent
		require.Len(t, req.Tools, 1)
		require.Len(t, req.Tools[0].FunctionDeclarations, 1)
		assert.Equal(t, "calculator", req.Tools[0].FunctionDeclarations[0].Name)

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "Done"}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewGeminiChatModel("key", "gemini-3.1-pro", WithGeminiBaseURL(ts.URL))
	clientWithTools, err := client.WithTools([]*schema.ToolInfo{
		{
			Name: "calculator",
			Desc: "Calculate math",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"expression": {Type: "string", Desc: "Math expression", Required: true},
			}),
		},
	})
	require.NoError(t, err)

	msg, err := clientWithTools.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "What is 2+2?"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Done", msg.Content)
}

func TestGeminiChatModel_Generate_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid API key"}`))
	}))
	defer ts.Close()

	client := NewGeminiChatModel("bad-key", "gemini-3.1-pro", WithGeminiBaseURL(ts.URL))

	_, err := client.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "Hello"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestGeminiChatModel_Stream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []geminiResponse{
			{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role:  "model",
							Parts: []geminiPart{{Text: "Hello"}},
						},
					},
				},
			},
			{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role:  "model",
							Parts: []geminiPart{{Text: " World"}},
						},
					},
				},
			},
		}

		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		}
	}))
	defer ts.Close()

	client := NewGeminiChatModel("key", "gemini-3.1-pro", WithGeminiBaseURL(ts.URL))

	sr, err := client.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "Hello"},
	})
	require.NoError(t, err)

	var allContent string
	for {
		msg, err := sr.Recv()
		if err != nil {
			break
		}
		allContent += msg.Content
	}

	assert.Equal(t, "Hello World", allContent)
}

func TestGeminiChatModel_WithTools_ReturnsCopy(t *testing.T) {
	original := NewGeminiChatModel("key", "gemini-3.1-pro")
	assert.Empty(t, original.tools)

	tools := []*schema.ToolInfo{
		{Name: "test_tool", Desc: "A test tool"},
	}

	withTools, err := original.WithTools(tools)
	require.NoError(t, err)

	// Original should not be modified
	assert.Empty(t, original.tools)

	// New client should have tools
	geminiWithTools := withTools.(*GeminiChatModel)
	require.Len(t, geminiWithTools.tools, 1)
	assert.Equal(t, "test_tool", geminiWithTools.tools[0].Name)
}

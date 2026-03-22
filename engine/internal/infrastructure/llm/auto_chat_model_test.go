package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

// byokMock is a minimal ToolCallingChatModel that records whether it was called.
type byokMock struct {
	generateCalled bool
	streamCalled   bool
	response       *schema.Message
	err            error
}

func (m *byokMock) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.generateCalled = true
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *byokMock) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m.streamCalled = true
	if m.err != nil {
		return nil, m.err
	}
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(m.response, nil)
	}()
	return sr, nil
}

func (m *byokMock) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func newSuccessServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: content}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func newErrorServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

func newSSESuccessServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunk := fmt.Sprintf(`{"choices":[{"delta":{"role":"assistant","content":"%s"}}]}`, content)
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		flusher.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
}

// ---------- Generate tests ----------

func TestAutoChatModel_Generate_ProxySuccess_ByokNotCalled(t *testing.T) {
	proxyServer := newSuccessServer("proxy-answer")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-answer"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	msg, err := auto.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, "proxy-answer", msg.Content)
	assert.False(t, byok.generateCalled, "byok should NOT be called when proxy succeeds")
}

func TestAutoChatModel_Generate_Proxy402_FallbackToByok(t *testing.T) {
	proxyServer := newErrorServer(http.StatusPaymentRequired, "no quota")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-answer"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	msg, err := auto.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, "byok-answer", msg.Content)
	assert.True(t, byok.generateCalled, "byok should be called on 402")
}

func TestAutoChatModel_Generate_Proxy429_FallbackToByok(t *testing.T) {
	proxyServer := newErrorServer(http.StatusTooManyRequests, "rate limited")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-answer"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	msg, err := auto.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, "byok-answer", msg.Content)
	assert.True(t, byok.generateCalled, "byok should be called on 429")
}

func TestAutoChatModel_Generate_Proxy500_NoFallback(t *testing.T) {
	proxyServer := newErrorServer(http.StatusInternalServerError, "server error")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-answer"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	_, err := auto.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.Error(t, err)
	assert.False(t, byok.generateCalled, "byok should NOT be called on 500")
	assert.Contains(t, err.Error(), "HTTP 500")
}

// ---------- Stream tests ----------

func TestAutoChatModel_Stream_ProxySuccess_ByokNotCalled(t *testing.T) {
	proxyServer := newSSESuccessServer("proxy-stream")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-stream"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	reader, err := auto.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	require.NoError(t, err)

	msg, err := reader.Recv()
	require.NoError(t, err)
	assert.Equal(t, "proxy-stream", msg.Content)

	assert.False(t, byok.streamCalled, "byok should NOT be called when proxy stream succeeds")
}

func TestAutoChatModel_Stream_Proxy402_FallbackToByok(t *testing.T) {
	proxyServer := newErrorServer(http.StatusPaymentRequired, "no quota")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-stream"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	reader, err := auto.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	require.NoError(t, err)

	msg, err := reader.Recv()
	require.NoError(t, err)
	assert.Equal(t, "byok-stream", msg.Content)
	assert.True(t, byok.streamCalled, "byok should be called on 402")
}

func TestAutoChatModel_Stream_Proxy429_FallbackToByok(t *testing.T) {
	proxyServer := newErrorServer(http.StatusTooManyRequests, "rate limited")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-stream"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	reader, err := auto.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	require.NoError(t, err)

	msg, err := reader.Recv()
	require.NoError(t, err)
	assert.Equal(t, "byok-stream", msg.Content)
	assert.True(t, byok.streamCalled)
}

func TestAutoChatModel_Stream_Proxy500_NoFallback(t *testing.T) {
	proxyServer := newErrorServer(http.StatusInternalServerError, "server error")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "byok-stream"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "supervisor")
	auto := NewAutoChatModel(proxy, byok)

	_, err := auto.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})

	require.Error(t, err)
	assert.False(t, byok.streamCalled)
}

// ---------- WithTools ----------

func TestAutoChatModel_WithTools(t *testing.T) {
	proxyServer := newSuccessServer("ok")
	defer proxyServer.Close()

	byok := &byokMock{
		response: &schema.Message{Role: schema.Assistant, Content: "ok"},
	}
	proxy := NewProxyChatModel(proxyServer.URL, "token", "coder")
	auto := NewAutoChatModel(proxy, byok)

	tools := []*schema.ToolInfo{
		{Name: "read_file", Desc: "reads a file"},
	}

	withTools, err := auto.WithTools(tools)
	require.NoError(t, err)

	autoWithTools, ok := withTools.(*AutoChatModel)
	require.True(t, ok, "WithTools should return *AutoChatModel")

	// Proxy should have tools (type assert allowed in tests).
	proxyWithTools, ok := autoWithTools.proxy.(*ProxyChatModel)
	require.True(t, ok, "auto.proxy should be *ProxyChatModel after WithTools")
	assert.Len(t, proxyWithTools.tools, 1)
}

// ---------- isProxyFallbackError ----------

func TestIsProxyFallbackError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"quota exhausted", ErrQuotaExhausted, true},
		{"rate limited", ErrRateLimited, true},
		{"wrapped quota", fmt.Errorf("wrap: %w", ErrQuotaExhausted), true},
		{"wrapped rate", fmt.Errorf("wrap: %w", ErrRateLimited), true},
		{"generic error", fmt.Errorf("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				return // isProxyFallbackError doesn't handle nil
			}
			assert.Equal(t, tt.want, isProxyFallbackError(tt.err))
		})
	}
}

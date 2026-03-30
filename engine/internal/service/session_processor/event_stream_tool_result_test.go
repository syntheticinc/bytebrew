package session_processor

import (
	"testing"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPublisher struct {
	events []*pb.SessionEvent
}

func (m *mockPublisher) PublishEvent(_ string, event *pb.SessionEvent) {
	m.events = append(m.events, event)
}

type mockStore struct{}

func (m *mockStore) Append(_, _ string, _ *pb.SessionEvent, _ map[string]interface{}) (int64, error) {
	return 1, nil
}

func TestSend_ToolResult_UsesFullResultFromMetadata(t *testing.T) {
	pub := &mockPublisher{}
	stream := NewEventStream("session-1", pub, &mockStore{})

	fullResult := "device1: iPhone 14 Pro\ndevice2: Pixel 8\ndevice3: Samsung Galaxy S24\ndevice4: OnePlus 12\ndevice5: Xiaomi 14"
	preview := "device1: iPhone 14 Pro..."

	err := stream.Send(&domain.AgentEvent{
		Type:    domain.EventTypeToolResult,
		Content: preview,
		Step:    1,
		Metadata: map[string]interface{}{
			"tool_name":   "device.list",
			"full_result": fullResult,
		},
	})

	require.NoError(t, err)
	require.Len(t, pub.events, 1)

	evt := pub.events[0]
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END, evt.Type)
	assert.Equal(t, fullResult, evt.Content, "Content should be the full result, not the truncated preview")
	assert.NotEqual(t, preview, evt.Content)
}

func TestSend_ToolResult_FallsBackToContent(t *testing.T) {
	pub := &mockPublisher{}
	stream := NewEventStream("session-1", pub, &mockStore{})

	content := "result without full_result metadata"

	err := stream.Send(&domain.AgentEvent{
		Type:    domain.EventTypeToolResult,
		Content: content,
		Step:    2,
		Metadata: map[string]interface{}{
			"tool_name": "device.list",
		},
	})

	require.NoError(t, err)
	require.Len(t, pub.events, 1)

	evt := pub.events[0]
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END, evt.Type)
	assert.Equal(t, content, evt.Content, "Content should fall back to event.Content when full_result is absent")
}

func TestSend_ToolResult_PreservesSummary(t *testing.T) {
	pub := &mockPublisher{}
	stream := NewEventStream("session-1", pub, &mockStore{})

	fullResult := "device1: iPhone 14 Pro\ndevice2: Pixel 8\ndevice3: Samsung Galaxy S24"
	summary := "3 devices found"

	err := stream.Send(&domain.AgentEvent{
		Type:    domain.EventTypeToolResult,
		Content: "device1: iPhone...",
		Step:    3,
		Metadata: map[string]interface{}{
			"tool_name":   "device.list",
			"full_result": fullResult,
			"summary":     summary,
		},
	})

	require.NoError(t, err)
	require.Len(t, pub.events, 1)

	evt := pub.events[0]
	assert.Equal(t, fullResult, evt.Content, "Content should be the full result")
	assert.Equal(t, summary, evt.ToolResultSummary, "ToolResultSummary should be the summary")
}

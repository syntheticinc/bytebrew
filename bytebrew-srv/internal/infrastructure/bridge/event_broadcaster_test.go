package bridge

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
)

// mockMessageSender captures messages sent to devices. Implements MessageSender.
type mockMessageSender struct {
	mu       sync.Mutex
	messages map[string][]*MobileMessage
}

func newMockMessageSender() *mockMessageSender {
	return &mockMessageSender{
		messages: make(map[string][]*MobileMessage),
	}
}

func (s *mockMessageSender) SendMessage(deviceID string, msg *MobileMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[deviceID] = append(s.messages[deviceID], msg)
	return nil
}

func (s *mockMessageSender) getMessages(deviceID string) []*MobileMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messages[deviceID]
}

// TC-EV-01: MessageCompleted — answer event serialized to flat format.
func TestSerializeEventForMobile_Answer(t *testing.T) {
	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "Hello world",
		AgentId: "supervisor",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "MessageCompleted", result["type"])
	assert.Equal(t, "Hello world", result["content"])
	assert.Equal(t, "assistant", result["role"])
	assert.Equal(t, "supervisor", result["agent_id"])
}

func TestSerializeEventForMobile_AnswerChunk(t *testing.T) {
	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK,
		Content: "partial",
		AgentId: "code-agent-1",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "StreamingProgress", result["type"])
	assert.Equal(t, "partial", result["content"])
}

// TC-EV-03: ToolExecutionStarted — tool start with arguments in flat format.
func TestSerializeEventForMobile_ToolStart(t *testing.T) {
	event := &pb.SessionEvent{
		Type:     pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START,
		CallId:   "call-1",
		ToolName: "read_file",
		ToolArguments: map[string]string{
			"path": "main.go",
		},
		AgentId: "supervisor",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "ToolExecutionStarted", result["type"])
	assert.Equal(t, "call-1", result["call_id"])
	assert.Equal(t, "read_file", result["tool_name"])

	args, ok := result["arguments"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "main.go", args["path"])
}

// TC-EV-04: ToolExecutionCompleted — tool end with result summary.
func TestSerializeEventForMobile_ToolEnd(t *testing.T) {
	event := &pb.SessionEvent{
		Type:              pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END,
		CallId:            "call-1",
		ToolName:          "read_file",
		ToolResultSummary: "50 lines",
		ToolHasError:      false,
		AgentId:           "supervisor",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "ToolExecutionCompleted", result["type"])
	assert.Equal(t, "50 lines", result["result_summary"])
	assert.Equal(t, false, result["has_error"])
}

// TC-EV-06: ReasoningChunk — reasoning content with agent_id.
func TestSerializeEventForMobile_Reasoning(t *testing.T) {
	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_REASONING,
		Content: "thinking...",
		AgentId: "supervisor",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "ReasoningChunk", result["type"])
	assert.Equal(t, "thinking...", result["content"])
	assert.Equal(t, "supervisor", result["agent_id"])
}

// TC-EV-05: AskUserRequested — question with options and agent_id.
func TestSerializeEventForMobile_AskUser(t *testing.T) {
	event := &pb.SessionEvent{
		Type:     pb.SessionEventType_SESSION_EVENT_ASK_USER,
		Question: "Continue?",
		Options:  []string{"yes", "no"},
		AgentId:  "supervisor",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "AskUserRequested", result["type"])
	assert.Equal(t, "Continue?", result["question"])
	assert.Equal(t, []string{"yes", "no"}, result["options"])
	assert.Equal(t, "supervisor", result["agent_id"])
}

// TC-EV-08: ProcessingStarted — state set to "processing".
func TestSerializeEventForMobile_ProcessingStarted(t *testing.T) {
	event := &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED,
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "ProcessingStarted", result["type"])
	assert.Equal(t, "processing", result["state"])
}

// TC-EV-09: ProcessingStopped — state set to "idle".
func TestSerializeEventForMobile_ProcessingStopped(t *testing.T) {
	event := &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED,
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "ProcessingStopped", result["type"])
	assert.Equal(t, "idle", result["state"])
}

// TC-EV-10: Error — error with message from ErrorDetail and code "error".
func TestSerializeEventForMobile_Error(t *testing.T) {
	event := &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_ERROR,
		ErrorDetail: &pb.Error{
			Message: "something broke",
		},
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "Error", result["type"])
	assert.Equal(t, "something broke", result["message"])
	assert.Equal(t, "error", result["code"])
}

// TC-EV-10 (edge case): Error with fallback to Content when ErrorDetail is nil.
func TestSerializeEventForMobile_ErrorFallbackContent(t *testing.T) {
	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ERROR,
		Content: "fallback error",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "fallback error", result["message"])
}

// TC-EV-07: PlanUpdated — plan name, steps with title/status, and agent_id.
func TestSerializeEventForMobile_PlanUpdate(t *testing.T) {
	event := &pb.SessionEvent{
		Type:     pb.SessionEventType_SESSION_EVENT_PLAN_UPDATE,
		PlanName: "Migration Plan",
		PlanSteps: []*pb.PlanStep{
			{Title: "Analyze", Status: "completed"},
			{Title: "Implement", Status: "in_progress"},
		},
		AgentId: "supervisor",
	}

	result := serializeEventForMobile(event)
	require.NotNil(t, result)
	assert.Equal(t, "PlanUpdated", result["type"])
	assert.Equal(t, "Migration Plan", result["plan_name"])
	assert.Equal(t, "supervisor", result["agent_id"])

	steps, ok := result["steps"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, steps, 2)
	assert.Equal(t, "Analyze", steps[0]["title"])
	assert.Equal(t, "completed", steps[0]["status"])
	assert.Equal(t, "Implement", steps[1]["title"])
	assert.Equal(t, "in_progress", steps[1]["status"])
}

func TestSerializeEventForMobile_UnknownType(t *testing.T) {
	event := &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_UNSPECIFIED,
	}

	result := serializeEventForMobile(event)
	assert.Nil(t, result)
}

// TC-B-14: Multi-device — 2 devices subscribed to same session → both receive events.
func TestEventBroadcaster_MultiDeviceBroadcast(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	// Subscribe two devices to the same session.
	broadcaster.Subscribe("device-1", "session-1", "")
	broadcaster.Subscribe("device-2", "session-1", "")

	// Broadcast an event for session-1.
	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "Hello from agent",
		AgentId: "supervisor",
	}
	broadcaster.BroadcastEvent("session-1", event)

	// Both devices must receive exactly one message.
	msgs1 := sender.getMessages("device-1")
	msgs2 := sender.getMessages("device-2")

	require.Len(t, msgs1, 1, "device-1 should receive one message")
	require.Len(t, msgs2, 1, "device-2 should receive one message")

	// Verify message structure for device-1.
	assert.Equal(t, "session_event", msgs1[0].Type)
	payload1 := msgs1[0].Payload
	assert.Equal(t, "session-1", payload1["session_id"])

	evt1, ok := payload1["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MessageCompleted", evt1["type"])
	assert.Equal(t, "Hello from agent", evt1["content"])

	// Verify message structure for device-2.
	assert.Equal(t, "session_event", msgs2[0].Type)
	payload2 := msgs2[0].Payload
	assert.Equal(t, "session-1", payload2["session_id"])

	evt2, ok := payload2["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MessageCompleted", evt2["type"])
	assert.Equal(t, "Hello from agent", evt2["content"])
}

func TestEventBroadcaster_OnlySubscribedSessionReceives(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	// device-1 subscribes to session-1, device-2 subscribes to session-2.
	broadcaster.Subscribe("device-1", "session-1", "")
	broadcaster.Subscribe("device-2", "session-2", "")

	// Broadcast to session-1 only.
	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "only for session-1",
		AgentId: "supervisor",
	}
	broadcaster.BroadcastEvent("session-1", event)

	msgs1 := sender.getMessages("device-1")
	msgs2 := sender.getMessages("device-2")

	require.Len(t, msgs1, 1, "device-1 should receive the event")
	assert.Empty(t, msgs2, "device-2 should NOT receive the event")
}

func TestEventBroadcaster_UnsubscribedDeviceDoesNotReceive(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	broadcaster.Subscribe("device-1", "session-1", "")
	broadcaster.Subscribe("device-2", "session-1", "")

	// Unsubscribe device-2 before broadcasting.
	broadcaster.Unsubscribe("device-2")

	event := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "after unsubscribe",
		AgentId: "supervisor",
	}
	broadcaster.BroadcastEvent("session-1", event)

	msgs1 := sender.getMessages("device-1")
	msgs2 := sender.getMessages("device-2")

	require.Len(t, msgs1, 1, "device-1 should receive the event")
	assert.Empty(t, msgs2, "device-2 should NOT receive after unsubscribe")
}

// TC-B-07: Subscribe + events — subscribe device → receive session events with correct structure.
func TestEventBroadcaster_SubscribeAndReceiveEvents(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	// Subscribe device.
	broadcaster.Subscribe("dev-1", "sess-1", "")

	// Broadcast multiple event types.
	events := []*pb.SessionEvent{
		{
			Type:    pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED,
			AgentId: "supervisor",
		},
		{
			Type:     pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START,
			CallId:   "c1",
			ToolName: "read_file",
			ToolArguments: map[string]string{"path": "main.go"},
			AgentId:  "supervisor",
		},
		{
			Type:              pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END,
			CallId:            "c1",
			ToolName:          "read_file",
			ToolResultSummary: "42 lines",
			AgentId:           "supervisor",
		},
		{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
			Content: "Done!",
			AgentId: "supervisor",
		},
	}

	for _, evt := range events {
		broadcaster.BroadcastEvent("sess-1", evt)
	}

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 4, "device should receive all 4 events")

	// Verify each message wraps a session_event with session_id and event_id.
	for i, msg := range msgs {
		assert.Equal(t, "session_event", msg.Type, "msg %d type", i)
		assert.Equal(t, "sess-1", msg.Payload["session_id"], "msg %d session_id", i)
		assert.NotEmpty(t, msg.Payload["event_id"], "msg %d event_id", i)

		evt, ok := msg.Payload["event"].(map[string]interface{})
		require.True(t, ok, "msg %d event should be map", i)
		assert.NotEmpty(t, evt["type"], "msg %d event type", i)
	}

	// Verify event types in order.
	evt0 := msgs[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "ProcessingStarted", evt0["type"])

	evt1 := msgs[1].Payload["event"].(map[string]interface{})
	assert.Equal(t, "ToolExecutionStarted", evt1["type"])
	assert.Equal(t, "read_file", evt1["tool_name"])

	evt2 := msgs[2].Payload["event"].(map[string]interface{})
	assert.Equal(t, "ToolExecutionCompleted", evt2["type"])
	assert.Equal(t, "42 lines", evt2["result_summary"])

	evt3 := msgs[3].Payload["event"].(map[string]interface{})
	assert.Equal(t, "MessageCompleted", evt3["type"])
	assert.Equal(t, "Done!", evt3["content"])
}

// Subscribe with empty lastEventID backfills ALL events from buffer.
func TestEventBroadcaster_BackfillsAllOnEmptyLastEventID(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	// Broadcast 3 events before any subscriber.
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED,
	})
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "hello",
		AgentId: "supervisor",
	})
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type: pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED,
	})

	// Subscribe with empty lastEventID → should receive ALL 3 events.
	broadcaster.Subscribe("dev-1", "sess-1", "")

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 3, "should backfill all 3 events on empty lastEventID")

	evt0 := msgs[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "ProcessingStarted", evt0["type"])
	assert.Equal(t, "mevt-1", msgs[0].Payload["event_id"])

	evt1 := msgs[1].Payload["event"].(map[string]interface{})
	assert.Equal(t, "MessageCompleted", evt1["type"])
	assert.Equal(t, "mevt-2", msgs[1].Payload["event_id"])

	evt2 := msgs[2].Payload["event"].(map[string]interface{})
	assert.Equal(t, "ProcessingStopped", evt2["type"])
	assert.Equal(t, "mevt-3", msgs[2].Payload["event_id"])
}

// Subscribe with empty lastEventID only backfills events for the subscribed session.
func TestEventBroadcaster_BackfillAllFiltersBySession(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "for sess-1",
		AgentId: "supervisor",
	})
	broadcaster.BroadcastEvent("sess-2", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "for sess-2",
		AgentId: "supervisor",
	})

	// Subscribe to sess-1 with empty lastEventID.
	broadcaster.Subscribe("dev-1", "sess-1", "")

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 1, "should only backfill sess-1 events")

	evt := msgs[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "for sess-1", evt["content"])
}

// TC-B-13: Backfill reconnect — subscribe with last_event_id → missed events replayed.
func TestEventBroadcaster_BackfillOnReconnect(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	// Broadcast 3 events to session-1 before any subscriber.
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED,
	})
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_REASONING,
		Content: "thinking...",
		AgentId: "supervisor",
	})
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "result",
		AgentId: "supervisor",
	})

	// No subscribers → no messages sent yet.
	assert.Empty(t, sender.getMessages("dev-1"))

	// Device subscribes with last_event_id = "mevt-1" (saw only event 1).
	// Should receive events 2 and 3 as backfill.
	broadcaster.Subscribe("dev-1", "sess-1", "mevt-1")

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 2, "should backfill 2 missed events after mevt-1")

	// Verify backfilled events.
	evt0 := msgs[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "ReasoningChunk", evt0["type"])
	assert.Equal(t, "mevt-2", msgs[0].Payload["event_id"])

	evt1 := msgs[1].Payload["event"].(map[string]interface{})
	assert.Equal(t, "MessageCompleted", evt1["type"])
	assert.Equal(t, "mevt-3", msgs[1].Payload["event_id"])
}

// TC-B-13: Backfill reconnect — subscribe with last_event_id filters by session.
func TestEventBroadcaster_BackfillFiltersBySession(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	// Broadcast events to different sessions.
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "event-1-sess-1",
		AgentId: "supervisor",
	})
	broadcaster.BroadcastEvent("sess-2", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "event-2-sess-2",
		AgentId: "supervisor",
	})
	broadcaster.BroadcastEvent("sess-1", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "event-3-sess-1",
		AgentId: "supervisor",
	})

	// Subscribe to sess-1 with last_event_id = "mevt-1" (saw only event 1).
	// Should receive only mevt-3 (sess-1), not mevt-2 (sess-2).
	broadcaster.Subscribe("dev-1", "sess-1", "mevt-1")

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 1, "should backfill only sess-1 events")

	evt := msgs[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "event-3-sess-1", evt["content"])
	assert.Equal(t, "mevt-3", msgs[0].Payload["event_id"])
}

// SendSessionStatus sends ProcessingStopped when not processing.
func TestEventBroadcaster_SendSessionStatus_Idle(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	broadcaster.SendSessionStatus("dev-1", "sess-1", false)

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 1)

	assert.Equal(t, "session_event", msgs[0].Type)
	// Synthetic events have empty event_id to bypass mobile dedup.
	assert.Empty(t, msgs[0].Payload["event_id"])
	evt, ok := msgs[0].Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ProcessingStopped", evt["type"])
	assert.Equal(t, "idle", evt["state"])
}

// SendSessionStatus sends ProcessingStarted when processing.
func TestEventBroadcaster_SendSessionStatus_Processing(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	broadcaster.SendSessionStatus("dev-1", "sess-1", true)

	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 1)

	evt, ok := msgs[0].Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ProcessingStarted", evt["type"])
	assert.Equal(t, "processing", evt["state"])
}

// TC-B-14: Multi-device — events only go to devices subscribed to the matching session.
func TestEventBroadcaster_MultiDevice_SessionIsolation(t *testing.T) {
	sender := newMockMessageSender()
	broadcaster := NewEventBroadcaster(sender)

	broadcaster.Subscribe("dev-1", "sess-A", "")
	broadcaster.Subscribe("dev-2", "sess-B", "")
	broadcaster.Subscribe("dev-3", "sess-A", "")

	broadcaster.BroadcastEvent("sess-A", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "for sess-A",
		AgentId: "supervisor",
	})
	broadcaster.BroadcastEvent("sess-B", &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "for sess-B",
		AgentId: "supervisor",
	})

	// dev-1 and dev-3: only sess-A event.
	msgs1 := sender.getMessages("dev-1")
	msgs3 := sender.getMessages("dev-3")
	require.Len(t, msgs1, 1)
	require.Len(t, msgs3, 1)

	evt1 := msgs1[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "for sess-A", evt1["content"])

	// dev-2: only sess-B event.
	msgs2 := sender.getMessages("dev-2")
	require.Len(t, msgs2, 1)
	evt2 := msgs2[0].Payload["event"].(map[string]interface{})
	assert.Equal(t, "for sess-B", evt2["content"])
}

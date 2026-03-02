package mobile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func TestEventBroadcaster_SubscribeAndBroadcast(t *testing.T) {
	b := NewEventBroadcaster(NewEventBuffer(0))

	ch, unsub := b.Subscribe("session-1", "sub-1")
	defer unsub()

	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswer,
		Timestamp: time.Now(),
		Content:   "Hello from agent",
		IsComplete: true,
	}

	b.Broadcast("session-1", event)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, "session-1", got.SessionId)
		assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_AGENT_MESSAGE, got.Type)

		msg := got.GetAgentMessage()
		require.NotNil(t, msg)
		assert.Equal(t, "Hello from agent", msg.Content)
		assert.True(t, msg.IsComplete)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewEventBroadcaster(NewEventBuffer(0))

	ch1, unsub1 := b.Subscribe("session-1", "sub-1")
	defer unsub1()
	ch2, unsub2 := b.Subscribe("session-1", "sub-2")
	defer unsub2()

	event := &domain.AgentEvent{
		Type:      domain.EventTypeReasoning,
		Timestamp: time.Now(),
		Content:   "Thinking...",
	}

	b.Broadcast("session-1", event)

	// Both subscribers should receive the event
	for _, ch := range []<-chan *pb.SessionEvent{ch1, ch2} {
		select {
		case got := <-ch:
			require.NotNil(t, got)
			assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_REASONING, got.Type)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	}
}

func TestEventBroadcaster_Unsubscribe(t *testing.T) {
	b := NewEventBroadcaster(NewEventBuffer(0))

	ch, unsub := b.Subscribe("session-1", "sub-1")
	unsub()

	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswer,
		Timestamp: time.Now(),
		Content:   "Should not be received",
	}

	b.Broadcast("session-1", event)

	select {
	case <-ch:
		t.Fatal("should not receive event after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// expected: no event
	}
}

func TestEventBroadcaster_BroadcastToWrongSession(t *testing.T) {
	b := NewEventBroadcaster(NewEventBuffer(0))

	ch, unsub := b.Subscribe("session-1", "sub-1")
	defer unsub()

	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswer,
		Timestamp: time.Now(),
		Content:   "Wrong session",
	}

	b.Broadcast("session-2", event) // different session

	select {
	case <-ch:
		t.Fatal("should not receive event from different session")
	case <-time.After(50 * time.Millisecond):
		// expected: no event
	}
}

func TestEventBroadcaster_HasSubscribers(t *testing.T) {
	b := NewEventBroadcaster(NewEventBuffer(0))

	assert.False(t, b.HasSubscribers("session-1"))

	_, unsub := b.Subscribe("session-1", "sub-1")
	assert.True(t, b.HasSubscribers("session-1"))

	unsub()
	assert.False(t, b.HasSubscribers("session-1"))
}

func TestEventBroadcaster_DoubleUnsubscribeIsSafe(t *testing.T) {
	b := NewEventBroadcaster(NewEventBuffer(0))

	_, unsub := b.Subscribe("session-1", "sub-1")
	unsub()
	unsub() // should not panic
}

func TestConvertEvent_ToolCall(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeToolCall,
		Timestamp: time.Now(),
		Step:      3,
		Content:   "read_file",
		Metadata: map[string]interface{}{
			"tool_name":          "read_file",
			"function_arguments": `{"path":"/src/main.go"}`,
		},
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_TOOL_CALL_START, got.Type)

	tc := got.GetToolCallStart()
	require.NotNil(t, tc)
	assert.Equal(t, "read_file", tc.ToolName)
	assert.Equal(t, "server-read_file-3", tc.CallId)
	assert.Equal(t, "/src/main.go", tc.Arguments["path"])
}

func TestConvertEvent_ToolResult(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeToolResult,
		Timestamp: time.Now(),
		Step:      3,
		Content:   "file contents...",
		Metadata: map[string]interface{}{
			"tool_name": "read_file",
			"summary":   "45 lines",
		},
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_TOOL_CALL_END, got.Type)

	tr := got.GetToolCallEnd()
	require.NotNil(t, tr)
	assert.Equal(t, "read_file", tr.ToolName)
	assert.Equal(t, "server-read_file-3", tr.CallId)
	assert.Equal(t, "45 lines", tr.ResultSummary)
	assert.False(t, tr.HasError)
}

func TestConvertEvent_UserQuestion(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeUserQuestion,
		Timestamp: time.Now(),
		Content:   "Should I proceed with the refactoring?",
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_ASK_USER, got.Type)

	ask := got.GetAskUser()
	require.NotNil(t, ask)
	assert.Equal(t, "Should I proceed with the refactoring?", ask.Question)
}

func TestConvertEvent_PlanProgress(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypePlanProgress,
		Timestamp: time.Now(),
		Content:   "Analyze codebase",
		Metadata: map[string]interface{}{
			"current_step": "Step 2: Review architecture",
			"progress":     "1/3",
		},
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_PLAN_UPDATE, got.Type)

	plan := got.GetPlan()
	require.NotNil(t, plan)
	assert.Equal(t, "Analyze codebase", plan.PlanName)
}

func TestConvertEvent_Error(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeError,
		Timestamp: time.Now(),
		Error: &domain.AgentError{
			Code:    "TIMEOUT",
			Message: "LLM request timed out",
		},
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_ERROR, got.Type)

	errEvent := got.GetErrorEvent()
	require.NotNil(t, errEvent)
	assert.Equal(t, "TIMEOUT", errEvent.Code)
	assert.Equal(t, "LLM request timed out", errEvent.Message)
}

func TestConvertEvent_ErrorWithoutErrorField(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeError,
		Timestamp: time.Now(),
		Content:   "something went wrong",
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)

	errEvent := got.GetErrorEvent()
	require.NotNil(t, errEvent)
	assert.Equal(t, "something went wrong", errEvent.Message)
}

func TestConvertEvent_AnswerChunk(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswerChunk,
		Timestamp: time.Now(),
		Content:   "partial answer",
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_ANSWER_CHUNK, got.Type)

	msg := got.GetAgentMessage()
	require.NotNil(t, msg)
	assert.Equal(t, "partial answer", msg.Content)
	assert.False(t, msg.IsComplete)
}

func TestConvertEvent_AgentSpawned(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeAgentSpawned,
		Timestamp: time.Now(),
		AgentID:   "code-agent-abc12345",
		Content:   "Starting code analysis",
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_SESSION_STATUS, got.Type)
	assert.Equal(t, "code-agent-abc12345", got.AgentId)

	status := got.GetSessionStatus()
	require.NotNil(t, status)
	assert.Contains(t, status.Message, "code-agent-abc12345")
	assert.Contains(t, status.Message, "Starting code analysis")
}

func TestConvertEvent_DefaultAgentID(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswer,
		Timestamp: time.Now(),
		Content:   "Hello",
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, "supervisor", got.AgentId)
}

func TestConvertEvent_PlanWithSteps(t *testing.T) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypePlanCreated,
		Timestamp: time.Now(),
		Content:   "Refactoring plan",
		Metadata: map[string]interface{}{
			"steps": []interface{}{
				map[string]interface{}{
					"description": "Analyze current code",
					"status":      "completed",
				},
				map[string]interface{}{
					"description": "Write new implementation",
					"status":      "in_progress",
				},
				map[string]interface{}{
					"description": "Add tests",
					"status":      "pending",
				},
			},
		},
	}

	got := convertEvent("session-1", event)
	require.NotNil(t, got)
	assert.Equal(t, pb.SessionEventType_SESSION_EVENT_TYPE_PLAN_UPDATE, got.Type)

	plan := got.GetPlan()
	require.NotNil(t, plan)
	assert.Equal(t, "Refactoring plan", plan.PlanName)
	require.Len(t, plan.Steps, 3)
	assert.Equal(t, "Analyze current code", plan.Steps[0].Title)
	assert.Equal(t, pb.PlanStepStatus_PLAN_STEP_STATUS_COMPLETED, plan.Steps[0].Status)
	assert.Equal(t, "Write new implementation", plan.Steps[1].Title)
	assert.Equal(t, pb.PlanStepStatus_PLAN_STEP_STATUS_IN_PROGRESS, plan.Steps[1].Status)
	assert.Equal(t, "Add tests", plan.Steps[2].Title)
	assert.Equal(t, pb.PlanStepStatus_PLAN_STEP_STATUS_PENDING, plan.Steps[2].Status)
}

func TestExtractToolArguments_ValidJSON(t *testing.T) {
	event := &domain.AgentEvent{
		Metadata: map[string]interface{}{
			"function_arguments": `{"path":"/src/main.go","line":42}`,
		},
	}

	args := extractToolArguments(event)
	assert.Equal(t, "/src/main.go", args["path"])
	assert.Equal(t, "42", args["line"])
}

func TestExtractToolArguments_InvalidJSON(t *testing.T) {
	event := &domain.AgentEvent{
		Metadata: map[string]interface{}{
			"function_arguments": `not-json`,
		},
	}

	args := extractToolArguments(event)
	assert.Equal(t, "not-json", args["_json"])
}

func TestExtractToolArguments_Empty(t *testing.T) {
	event := &domain.AgentEvent{
		Metadata: map[string]interface{}{},
	}

	args := extractToolArguments(event)
	assert.Empty(t, args)
}

func TestExtractToolArguments_BoolAndArray(t *testing.T) {
	event := &domain.AgentEvent{
		Metadata: map[string]interface{}{
			"function_arguments": `{"recursive":true,"files":["a.go","b.go"]}`,
		},
	}

	args := extractToolArguments(event)
	assert.Equal(t, "true", args["recursive"])
	assert.Equal(t, "a.go\nb.go", args["files"])
}

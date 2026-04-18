package ws

import (
	"log/slog"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
)

// serializeEvent converts a proto SessionEvent to a flat JSON map.
func serializeEvent(event *pb.SessionEvent) map[string]interface{} {
	switch event.GetType() {
	case pb.SessionEventType_SESSION_EVENT_ANSWER:
		return map[string]interface{}{
			"type":     "MessageCompleted",
			"content":  event.GetContent(),
			"role":     "assistant",
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK:
		return map[string]interface{}{
			"type":     "StreamingProgress",
			"content":  event.GetContent(),
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START:
		args := make(map[string]interface{}, len(event.GetToolArguments()))
		for k, v := range event.GetToolArguments() {
			args[k] = v
		}
		return map[string]interface{}{
			"type":      "ToolExecutionStarted",
			"call_id":   event.GetCallId(),
			"tool_name": event.GetToolName(),
			"arguments": args,
			"agent_id":  event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END:
		return map[string]interface{}{
			"type":           "ToolExecutionCompleted",
			"call_id":        event.GetCallId(),
			"tool_name":      event.GetToolName(),
			"result":         event.GetContent(),
			"result_summary": event.GetToolResultSummary(),
			"has_error":      event.GetToolHasError(),
			"agent_id":       event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_REASONING:
		return map[string]interface{}{
			"type":     "ReasoningChunk",
			"content":  event.GetContent(),
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_ASK_USER:
		return map[string]interface{}{
			"type":     "AskUserRequested",
			"question": event.GetQuestion(),
			"call_id":  event.GetCallId(),
			"options":  event.GetOptions(),
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED:
		return map[string]interface{}{
			"type":  "ProcessingStarted",
			"state": "processing",
		}

	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED:
		return map[string]interface{}{
			"type":  "ProcessingStopped",
			"state": "idle",
		}

	case pb.SessionEventType_SESSION_EVENT_ERROR:
		msg := event.GetContent()
		if detail := event.GetErrorDetail(); detail != nil {
			msg = detail.GetMessage()
		}
		return map[string]interface{}{
			"type":    "Error",
			"message": msg,
			"code":    "error",
		}

	case pb.SessionEventType_SESSION_EVENT_USER_MESSAGE:
		return map[string]interface{}{
			"type":    "UserMessage",
			"content": event.GetContent(),
			"role":    "user",
		}

	case pb.SessionEventType_SESSION_EVENT_PLAN_UPDATE:
		steps := make([]map[string]interface{}, 0, len(event.GetPlanSteps()))
		for _, s := range event.GetPlanSteps() {
			steps = append(steps, map[string]interface{}{
				"title":  s.GetTitle(),
				"status": s.GetStatus(),
			})
		}
		return map[string]interface{}{
			"type":      "PlanUpdated",
			"plan_name": event.GetPlanName(),
			"steps":     steps,
			"agent_id":  event.GetAgentId(),
		}

	default:
		slog.Warn("[WS] unknown session event type", "type", event.GetType().String())
		return nil
	}
}

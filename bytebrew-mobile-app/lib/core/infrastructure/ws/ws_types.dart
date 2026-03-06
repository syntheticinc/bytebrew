import 'dart:typed_data';

import 'package:bytebrew_mobile/core/domain/mobile_session.dart';

// ---------------------------------------------------------------------------
// DTOs
// ---------------------------------------------------------------------------

/// Result of a ping request.
class PingResult {
  const PingResult({
    required this.timestamp,
    required this.serverName,
    required this.serverId,
  });

  final DateTime timestamp;
  final String serverName;
  final String serverId;
}

/// Result of a pair request.
class PairResult {
  PairResult({
    required this.deviceId,
    required this.deviceToken,
    required this.serverName,
    required this.serverId,
    this.serverPublicKey,
  });

  final String deviceId;
  final String deviceToken;
  final String serverName;
  final String serverId;
  final Uint8List? serverPublicKey;
}

/// Result of a command request (sendNewTask, sendAskUserReply, cancelSession).
class SendCommandResult {
  const SendCommandResult({required this.success, this.errorMessage = ''});

  final bool success;
  final String errorMessage;
}

/// Result of a listSessions request.
class ListSessionsResult {
  const ListSessionsResult({
    required this.sessions,
    required this.serverName,
    required this.serverId,
  });

  final List<MobileSession> sessions;
  final String serverName;
  final String serverId;
}

// ---------------------------------------------------------------------------
// Session event types
// ---------------------------------------------------------------------------

/// The type of a session event received from the server.
enum SessionEventType {
  unspecified,
  agentMessage,
  toolCallStart,
  toolCallEnd,
  reasoning,
  askUser,
  plan,
  sessionStatus,
  error,
}

/// A single event from a session subscription stream.
class SessionEvent {
  SessionEvent({
    required this.eventId,
    required this.sessionId,
    required this.type,
    required this.timestamp,
    this.agentId = '',
    this.step = 0,
    this.payload,
  });

  final String eventId;
  final String sessionId;
  final SessionEventType type;
  final DateTime timestamp;
  final String agentId;
  final int step;
  final SessionEventPayload? payload;
}

// ---------------------------------------------------------------------------
// Payload hierarchy
// ---------------------------------------------------------------------------

/// Base class for session event payloads.
sealed class SessionEventPayload {
  const SessionEventPayload();
}

/// Payload for an agent message event.
class AgentMessagePayload extends SessionEventPayload {
  const AgentMessagePayload({required this.content, required this.isComplete});

  final String content;
  final bool isComplete;
}

/// Payload for a tool call start event.
class ToolCallStartPayload extends SessionEventPayload {
  const ToolCallStartPayload({
    required this.callId,
    required this.toolName,
    required this.arguments,
  });

  final String callId;
  final String toolName;
  final Map<String, String> arguments;
}

/// Payload for a tool call end event.
class ToolCallEndPayload extends SessionEventPayload {
  const ToolCallEndPayload({
    required this.callId,
    required this.toolName,
    required this.resultSummary,
    required this.hasError,
  });

  final String callId;
  final String toolName;
  final String resultSummary;
  final bool hasError;
}

/// Payload for a reasoning event.
class ReasoningPayload extends SessionEventPayload {
  const ReasoningPayload({required this.content, required this.isComplete});

  final String content;
  final bool isComplete;
}

/// Payload for an ask-user event.
class AskUserPayload extends SessionEventPayload {
  const AskUserPayload({
    required this.question,
    required this.options,
    required this.isAnswered,
  });

  final String question;
  final List<String> options;
  final bool isAnswered;
}

/// Payload for a plan event.
class PlanPayload extends SessionEventPayload {
  const PlanPayload({required this.planName, required this.steps});

  final String planName;
  final List<PlanStepPayload> steps;
}

/// A single step within a plan.
class PlanStepPayload {
  const PlanStepPayload({required this.title, required this.status});

  final String title;
  final WsPlanStepStatus status;
}

/// Status of a plan step.
enum WsPlanStepStatus { unspecified, pending, inProgress, completed, failed }

/// Payload for a session status change event.
class SessionStatusPayload extends SessionEventPayload {
  const SessionStatusPayload({required this.state, required this.message});

  final MobileSessionState state;
  final String message;
}

/// Payload for an error event.
class ErrorPayload extends SessionEventPayload {
  const ErrorPayload({required this.code, required this.message});

  final String code;
  final String message;
}

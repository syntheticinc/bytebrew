import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/message_mapper.dart';

/// Parses a CLI event JSON into a [ChatMessage], or null for status-only events.
ChatMessage? parseEventToChatMessage(Map<String, dynamic> event) {
  final type = event['type'] as String?;

  switch (type) {
    case 'MessageCompleted':
      final message = event['message'] as Map<String, dynamic>?;
      if (message == null) return null;
      return MessageMapper.fromSnapshot(message);

    case 'ToolExecutionStarted':
      final exec = event['execution'] as Map<String, dynamic>?;
      if (exec == null) return null;
      return _parseToolExecution(exec, isComplete: false);

    case 'ToolExecutionCompleted':
      final exec = event['execution'] as Map<String, dynamic>?;
      if (exec == null) return null;
      return _parseToolExecution(exec, isComplete: true);

    case 'AskUserRequested':
      return _parseAskUserEvent(event);

    case 'ErrorOccurred':
      return ChatMessage(
        id: 'error-${DateTime.now().millisecondsSinceEpoch}',
        type: ChatMessageType.systemMessage,
        content: event['message'] as String? ?? 'Unknown error',
        timestamp: DateTime.now(),
      );

    default:
      return null;
  }
}

ChatMessage _parseToolExecution(
  Map<String, dynamic> exec, {
  required bool isComplete,
}) {
  final callId = exec['callId'] as String? ?? '';
  final rawArgs = exec['arguments'];
  final arguments = <String, String>{};
  if (rawArgs is Map) {
    for (final e in rawArgs.entries) {
      arguments[e.key.toString()] = e.value.toString();
    }
  }

  return ChatMessage(
    id: callId,
    type: isComplete ? ChatMessageType.toolResult : ChatMessageType.toolCall,
    content: '',
    timestamp: DateTime.now(),
    toolCall: ToolCallData(
      id: callId,
      toolName: exec['toolName'] as String? ?? '',
      arguments: arguments,
      status: isComplete
          ? (exec['error'] != null
                ? ToolCallStatus.failed
                : ToolCallStatus.completed)
          : ToolCallStatus.running,
      result: exec['summary'] as String? ?? exec['result'] as String?,
      error: exec['error'] as String?,
    ),
    agentId: exec['agentId'] as String?,
  );
}

ChatMessage? _parseAskUserEvent(Map<String, dynamic> event) {
  final questions = event['questions'] as List<dynamic>?;
  if (questions == null || questions.isEmpty) return null;

  final q = questions.first as Map<String, dynamic>;
  final id = 'ask-${DateTime.now().millisecondsSinceEpoch}';
  final question = q['question'] as String? ?? '';
  final options =
      (q['options'] as List<dynamic>?)?.map((o) => o.toString()).toList() ?? [];

  return ChatMessage(
    id: id,
    type: ChatMessageType.askUser,
    content: question,
    timestamp: DateTime.now(),
    askUser: AskUserData(
      id: id,
      question: question,
      options: options,
      status: AskUserStatus.pending,
    ),
  );
}

/// Parses an AgentLifecycle event into [AgentInfo].
AgentInfo? parseAgentLifecycle(Map<String, dynamic> event) {
  if (event['type'] != 'AgentLifecycle') return null;

  final agentId = event['agentId'] as String? ?? '';
  final lifecycle = event['lifecycle'] as String? ?? '';

  final status = switch (lifecycle) {
    'spawned' => AgentStatus.running,
    'completed' => AgentStatus.completed,
    'failed' => AgentStatus.failed,
    _ => AgentStatus.running,
  };

  return AgentInfo(
    agentId: agentId,
    status: status,
    description: event['description'] as String? ?? '',
    lastActivityAt: DateTime.now(),
  );
}

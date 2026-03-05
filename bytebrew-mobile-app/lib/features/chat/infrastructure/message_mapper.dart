import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';

/// Maps CLI MessageSnapshot JSON to Flutter [ChatMessage] domain entities.
class MessageMapper {
  const MessageMapper._();

  /// Converts a CLI MessageSnapshot JSON map into a [ChatMessage].
  static ChatMessage fromSnapshot(Map<String, dynamic> json) {
    final id = json['id'] as String? ?? '';
    final content = json['content'] as String? ?? '';
    final timestamp = _parseTimestamp(json['timestamp']);

    final type = _resolveType(json);

    return ChatMessage(
      id: id,
      type: type,
      content: content,
      timestamp: timestamp,
      toolCall: _parseToolCall(json['toolCall'] as Map<String, dynamic>?),
      plan: _parsePlan(json['plan'] as Map<String, dynamic>?),
      askUser: _parseAskUser(json['askUser'] as Map<String, dynamic>?),
      agentId: json['agentId'] as String?,
    );
  }

  static ChatMessageType _resolveType(Map<String, dynamic> json) {
    final role = json['role'] as String? ?? '';
    final hasToolCall = json['toolCall'] != null;
    final hasToolResult = json['toolResult'] != null;
    final hasReasoning = json['reasoning'] != null;
    final hasPlan = json['plan'] != null;
    final hasAskUser = json['askUser'] != null;

    if (hasAskUser) return ChatMessageType.askUser;
    if (hasPlan) return ChatMessageType.planUpdate;

    if (role == 'tool') {
      if (hasToolCall) return ChatMessageType.toolCall;
      if (hasToolResult) return ChatMessageType.toolResult;
      return ChatMessageType.toolResult;
    }

    if (role == 'assistant') {
      if (hasToolCall) return ChatMessageType.toolCall;
      if (hasReasoning) return ChatMessageType.reasoning;
      return ChatMessageType.agentMessage;
    }

    if (role == 'user') return ChatMessageType.userMessage;
    if (role == 'system') return ChatMessageType.systemMessage;

    return ChatMessageType.systemMessage;
  }

  static ToolCallData? _parseToolCall(Map<String, dynamic>? tc) {
    if (tc == null) return null;

    final rawArgs = tc['arguments'];
    final arguments = <String, String>{};
    if (rawArgs is Map) {
      for (final entry in rawArgs.entries) {
        arguments[entry.key.toString()] = entry.value.toString();
      }
    }

    return ToolCallData(
      id: tc['id'] as String? ?? '',
      toolName: tc['toolName'] as String? ?? tc['name'] as String? ?? '',
      arguments: arguments,
      status: _parseToolCallStatus(tc['status'] as String?),
      result: tc['result'] as String?,
      fullResult: tc['fullResult'] as String?,
      error: tc['error'] as String?,
    );
  }

  static ToolCallStatus _parseToolCallStatus(String? status) {
    return switch (status) {
      'completed' => ToolCallStatus.completed,
      'failed' => ToolCallStatus.failed,
      _ => ToolCallStatus.running,
    };
  }

  static PlanData? _parsePlan(Map<String, dynamic>? plan) {
    if (plan == null) return null;

    final rawSteps = plan['steps'] as List<dynamic>? ?? [];
    final steps = rawSteps.indexed.map((entry) {
      final step = entry.$2 as Map<String, dynamic>;
      final rawCompletedAt = step['completedAt'];
      final completedAt = rawCompletedAt is String
          ? DateTime.tryParse(rawCompletedAt)
          : null;

      return PlanStep(
        index: entry.$1,
        description: step['description'] as String? ?? '',
        status: _parsePlanStepStatus(step['status'] as String?),
        completedAt: completedAt,
      );
    }).toList();

    return PlanData(goal: plan['goal'] as String? ?? '', steps: steps);
  }

  static PlanStepStatus _parsePlanStepStatus(String? status) {
    return switch (status) {
      'completed' => PlanStepStatus.completed,
      'in_progress' || 'inProgress' => PlanStepStatus.inProgress,
      _ => PlanStepStatus.pending,
    };
  }

  static AskUserData? _parseAskUser(Map<String, dynamic>? askUser) {
    if (askUser == null) return null;

    final rawOptions = askUser['options'] as List<dynamic>? ?? [];
    final options = rawOptions.map((o) => o.toString()).toList();

    return AskUserData(
      id: askUser['id'] as String? ?? '',
      question: askUser['question'] as String? ?? '',
      options: options,
      status: _parseAskUserStatus(askUser['status'] as String?),
      answer: askUser['answer'] as String?,
    );
  }

  static AskUserStatus _parseAskUserStatus(String? status) {
    return switch (status) {
      'answered' => AskUserStatus.answered,
      _ => AskUserStatus.pending,
    };
  }

  /// Parses a timestamp value that may be an [int] (milliseconds since epoch)
  /// or a [String] (ISO 8601). Falls back to [DateTime.now] if the value is
  /// null or an unrecognised type.
  static DateTime _parseTimestamp(dynamic raw) {
    if (raw is int) {
      return DateTime.fromMillisecondsSinceEpoch(raw);
    }
    if (raw is String) {
      return DateTime.tryParse(raw) ?? DateTime.now();
    }
    return DateTime.now();
  }
}

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';

/// Type of a chat message in the conversation.
enum ChatMessageType {
  userMessage,
  agentMessage,
  toolCall,
  toolResult,
  planUpdate,
  askUser,
  reasoning,
  systemMessage,
}

/// A single message in a chat session.
class ChatMessage {
  const ChatMessage({
    required this.id,
    required this.type,
    required this.content,
    required this.timestamp,
    this.toolCall,
    this.plan,
    this.askUser,
    this.agentId,
  });

  final String id;
  final ChatMessageType type;
  final String content;
  final DateTime timestamp;
  final ToolCallData? toolCall;
  final PlanData? plan;
  final AskUserData? askUser;
  final String? agentId;

  ChatMessage copyWith({
    String? id,
    ChatMessageType? type,
    String? content,
    DateTime? timestamp,
    ToolCallData? toolCall,
    PlanData? plan,
    AskUserData? askUser,
    String? agentId,
  }) {
    return ChatMessage(
      id: id ?? this.id,
      type: type ?? this.type,
      content: content ?? this.content,
      timestamp: timestamp ?? this.timestamp,
      toolCall: toolCall ?? this.toolCall,
      plan: plan ?? this.plan,
      askUser: askUser ?? this.askUser,
      agentId: agentId ?? this.agentId,
    );
  }
}

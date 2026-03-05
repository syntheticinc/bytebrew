import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';

/// Repository for chat interactions within a session.
abstract class ChatRepository {
  /// Returns all messages for the given [sessionId].
  Future<List<ChatMessage>> getMessages(String sessionId);

  /// Sends a user [text] message to the given [sessionId].
  Future<void> sendMessage(String sessionId, String text);

  /// Answers an ask-user prompt in the given session.
  Future<void> answerAskUser(String sessionId, String askUserId, String answer);

  /// Cancels the current operation in the given session.
  Future<void> cancel(String sessionId);

  /// Returns a stream of messages for real-time updates.
  /// Returns null if not supported (e.g. mock/offline mode).
  Stream<List<ChatMessage>>? watchMessages();

  /// Returns a stream of agent info for multi-agent sessions.
  /// Returns null if not supported (e.g. mock/offline mode).
  Stream<List<AgentInfo>>? watchAgents();
}

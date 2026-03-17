import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';

/// [ChatRepository] that always returns empty data.
///
/// Used when no WebSocket connection is active.
class EmptyChatRepository implements ChatRepository {
  const EmptyChatRepository();

  @override
  Future<List<ChatMessage>> getMessages(String sessionId) async => [];

  @override
  Stream<List<ChatMessage>>? watchMessages() => null;

  @override
  Future<void> sendMessage(String sessionId, String text) async {}

  @override
  Future<void> answerAskUser(
    String sessionId,
    String askUserId,
    String answer,
  ) async {}

  @override
  Future<void> cancel(String sessionId) async {}

  @override
  Stream<List<AgentInfo>>? watchAgents() => null;

  @override
  Stream<bool>? watchProcessing() => null;
}

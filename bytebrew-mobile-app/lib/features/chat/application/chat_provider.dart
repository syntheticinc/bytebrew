import 'dart:async';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/empty_chat_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'chat_provider.g.dart';

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Returns the WS-backed repository when connected, [EmptyChatRepository]
/// otherwise.
@riverpod
ChatRepository sessionChatRepository(Ref ref, String sessionId) {
  final status = ref.watch(wsConnectionProvider);
  if (status != WsConnectionStatus.connected) {
    return const EmptyChatRepository();
  }
  final notifier = ref.read(wsConnectionProvider.notifier);
  final repo = notifier.repository;
  return repo ?? const EmptyChatRepository();
}

/// Manages chat messages for a given session.
///
/// Automatically selects the right repository implementation based on the
/// WebSocket connection. Subscribes to real-time message streams when
/// available.
@riverpod
class ChatMessages extends _$ChatMessages {
  StreamSubscription<List<ChatMessage>>? _subscription;

  @override
  FutureOr<List<ChatMessage>> build(String sessionId) async {
    final repo = ref.watch(sessionChatRepositoryProvider(sessionId));

    // Clean up previous subscription on rebuild.
    _subscription?.cancel();

    final messageStream = repo.watchMessages();
    if (messageStream != null) {
      _subscription = messageStream.listen((messages) {
        state = AsyncData(messages);
      });
      ref.onDispose(() => _subscription?.cancel());
    }

    return repo.getMessages(sessionId);
  }

  /// Sends a user [text] message to the current session.
  Future<void> sendMessage(String text) async {
    final repo = ref.read(sessionChatRepositoryProvider(sessionId));
    await repo.sendMessage(sessionId, text);
    state = AsyncData(await repo.getMessages(sessionId));
  }

  /// Answers a pending ask-user prompt.
  Future<void> answerAskUser(String askUserId, String answer) async {
    final repo = ref.read(sessionChatRepositoryProvider(sessionId));
    await repo.answerAskUser(sessionId, askUserId, answer);

    // For repos without streaming, re-fetch. Stream-based repos get updates
    // automatically via the watchMessages() subscription.
    if (repo.watchMessages() == null) {
      state = AsyncData(await repo.getMessages(sessionId));
    }
  }

  /// Cancels the current operation.
  Future<void> cancel() async {
    final repo = ref.read(sessionChatRepositoryProvider(sessionId));
    await repo.cancel(sessionId);
    state = AsyncData(await repo.getMessages(sessionId));
  }
}

/// Returns the active plan from the latest planUpdate message, or null.
@riverpod
PlanData? activePlan(Ref ref, String sessionId) {
  final messagesAsync = ref.watch(chatMessagesProvider(sessionId));
  return messagesAsync.whenOrNull(
    data: (messages) {
      final planMessages = messages.where(
        (m) => m.type == ChatMessageType.planUpdate && m.plan != null,
      );
      if (planMessages.isEmpty) {
        return null;
      }
      return planMessages.last.plan;
    },
  );
}

/// Returns the pending ask-user message, or null if none.
@riverpod
ChatMessage? pendingAskUser(Ref ref, String sessionId) {
  final messagesAsync = ref.watch(chatMessagesProvider(sessionId));
  return messagesAsync.whenOrNull(
    data: (messages) {
      final askUsers = messages.where(
        (m) =>
            m.type == ChatMessageType.askUser &&
            m.askUser?.status == AskUserStatus.pending,
      );
      if (askUsers.isEmpty) {
        return null;
      }
      return askUsers.last;
    },
  );
}

import 'dart:async';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/infrastructure/storage/chat_message_store.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/empty_chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'chat_provider.g.dart';

/// Singleton [ChatMessageStore] for persisting chat messages to SQLite.
///
/// Kept alive to avoid re-opening the database on every chat screen visit.
@Riverpod(keepAlive: true)
ChatMessageStore chatMessageStore(Ref ref) {
  final store = ChatMessageStore();
  ref.onDispose(store.close);
  return store;
}

/// Provides a default [ChatRepository].
///
/// Returns [EmptyChatRepository] by default. Session-specific repositories
/// are created via [sessionChatRepositoryProvider].
@riverpod
ChatRepository chatRepository(Ref ref) {
  return const EmptyChatRepository();
}

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Uses [ref.read] instead of [ref.watch] on [sessionsProvider] to avoid
/// rebuilding (and dropping messages) when the session list refreshes.
/// The serverId is captured once at creation time.
/// Falls back to [EmptyChatRepository] if the session is not found.
@riverpod
ChatRepository sessionChatRepository(Ref ref, String sessionId) {
  final sessionsAsync = ref.read(sessionsProvider);
  final session = sessionsAsync.whenOrNull(
    data: (sessions) => sessions.where((s) => s.id == sessionId).firstOrNull,
  );
  if (session == null) return const EmptyChatRepository();

  final manager = ref.read(connectionManagerProvider);
  final store = ref.read(chatMessageStoreProvider);
  final repo = WsChatRepository(
    connectionManager: manager,
    serverId: session.serverId,
    sessionId: sessionId,
    messageStore: store,
  );
  repo.subscribe();
  ref.onDispose(repo.dispose);
  return repo;
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
    if (messageStream == null) {
      return repo.getMessages(sessionId);
    }

    // Subscribe to live updates — each emission replaces the state.
    _subscription = messageStream.listen((messages) {
      state = AsyncData(messages);
    });
    ref.onDispose(() => _subscription?.cancel());

    // If messages are already loaded (rebuild after reconnect), return them.
    final existing = await repo.getMessages(sessionId);
    if (existing.isNotEmpty) return existing;

    // Otherwise wait for the first stream event (history backfill) so the
    // provider stays in AsyncLoading until real data arrives, preventing
    // the empty-state flash. Timeout prevents infinite spinner when the
    // session genuinely has no history.
    try {
      return await messageStream.first.timeout(const Duration(seconds: 3));
    } on TimeoutException {
      return const [];
    }
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

/// Whether the session is currently being processed by the server.
///
/// Emits `true` when a ProcessingStarted event is received and `false`
/// when processing stops (idle / completed / failed).
@riverpod
Stream<bool> isProcessing(Ref ref, String sessionId) {
  final repo = ref.watch(sessionChatRepositoryProvider(sessionId));
  final stream = repo.watchProcessing();
  if (stream == null) return const Stream.empty();
  return stream;
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

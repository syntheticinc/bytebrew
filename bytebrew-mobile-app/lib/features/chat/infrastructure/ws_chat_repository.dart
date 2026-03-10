import 'dart:async';

import 'package:flutter/foundation.dart';

import 'package:bytebrew_mobile/core/domain/agent_info.dart';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/infrastructure/storage/chat_message_store.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';

/// [ChatRepository] implementation backed by WebSocket via [WsConnectionManager].
///
/// Manages an internal message list, handles optimistic updates for user
/// actions, and processes session events from the server subscription.
class WsChatRepository implements ChatRepository {
  static const _maxMessages = 100;
  WsChatRepository({
    required WsConnectionManager connectionManager,
    required String serverId,
    required String sessionId,
    ChatMessageStore? messageStore,
  }) : _connectionManager = connectionManager,
       _serverId = serverId,
       _sessionId = sessionId,
       _messageStore = messageStore;

  final WsConnectionManager _connectionManager;
  final String _serverId;
  final String _sessionId;
  final ChatMessageStore? _messageStore;

  /// Whether messages have been loaded from the local DB.
  bool _dbLoaded = false;

  final List<ChatMessage> _messages = [];
  final StreamController<List<ChatMessage>> _messageController =
      StreamController<List<ChatMessage>>.broadcast();
  final StreamController<bool> _processingController =
      StreamController<bool>.broadcast();
  bool _isProcessing = false;

  final List<AgentInfo> _agents = [];
  final StreamController<List<AgentInfo>> _agentController =
      StreamController<List<AgentInfo>>.broadcast();

  /// Tracks seen event IDs to deduplicate backfill events after reconnect.
  final _seenEventIds = <String>{};
  static const _maxSeenIds = 500;

  StreamSubscription<SessionEvent>? _subscription;
  String? _lastEventId;
  bool _disposed = false;

  /// Whether the first batch of events has been received from subscribe.
  /// Used by the UI to show a loader instead of empty state while waiting.
  bool _historyLoaded = false;
  bool get isHistoryLoaded => _historyLoaded;

  /// Debounce timer for _emitMessages to batch rapid backfill events.
  Timer? _emitDebounce;

  VoidCallback? _connectionListener;

  /// Tracks the last observed connection status to detect actual transitions.
  WsConnectionStatus? _lastConnectionStatus;

  /// Stable ID for send-timeout errors so they can be removed retroactively
  /// when session events confirm the task was actually received.
  static const _sendErrorId = 'send-timeout-error';

  /// Whether this repository is actively subscribed to session events.
  bool get isSubscribed => _subscription != null;

  // ---------------------------------------------------------------------------
  // ChatRepository
  // ---------------------------------------------------------------------------

  /// Loads messages from the local database if not yet loaded.
  ///
  /// Called lazily on first [getMessages] and before [subscribe] so that
  /// the in-memory list is seeded from persistent storage.
  Future<void> _loadFromDb() async {
    if (_dbLoaded || _messageStore == null) return;
    _dbLoaded = true;

    try {
      final stored = await _messageStore.getMessages(_sessionId);
      if (stored.isEmpty) return;

      // Merge stored messages into in-memory list without duplicates.
      for (final msg in stored) {
        final idx = _messages.indexWhere((m) => m.id == msg.id);
        if (idx == -1) {
          _messages.add(msg);
        }
      }
    } catch (e) {
      debugPrint('[WsChatRepository] Failed to load from DB: $e');
    }
  }

  @override
  Future<List<ChatMessage>> getMessages(String sessionId) async {
    await _loadFromDb();
    return List.unmodifiable(_messages);
  }

  @override
  Future<void> sendMessage(String sessionId, String text) async {
    final userMessage = ChatMessage(
      id: 'user-${DateTime.now().millisecondsSinceEpoch}',
      type: ChatMessageType.userMessage,
      content: text,
      timestamp: DateTime.now(),
    );
    _upsertMessage(userMessage);
    _emitMessages();

    // If the connection exists but is reconnecting, wait for it.
    var connection = _connectionManager.getConnection(_serverId);
    if (connection != null &&
        connection.status != WsConnectionStatus.connected) {
      connection = await _waitForConnection();
    }

    if (connection == null ||
        connection.status != WsConnectionStatus.connected) {
      _upsertMessage(
        ChatMessage(
          id: 'error-${DateTime.now().millisecondsSinceEpoch}',
          type: ChatMessageType.systemMessage,
          content: 'Failed to send: Server not connected',
          timestamp: DateTime.now(),
        ),
      );
      _emitMessages();
      return;
    }

    final result = await connection.client.sendNewTask(
      deviceToken: connection.server.deviceToken!,
      sessionId: _sessionId,
      task: text,
    );

    if (!result.success) {
      // Timeout errors for new_task are usually harmless: the CLI received
      // the task but the ack didn't make it back through a dead connection.
      // Use a stable ID so _handleSessionStatus can remove this error when
      // ProcessingStarted arrives, confirming the task was received.
      final isTimeout = result.errorMessage.contains('timed out');

      final errorText = result.errorMessage.isEmpty
          ? 'Server not connected'
          : result.errorMessage;
      _upsertMessage(
        ChatMessage(
          id: isTimeout
              ? _sendErrorId
              : 'error-${DateTime.now().millisecondsSinceEpoch}',
          type: ChatMessageType.systemMessage,
          content: 'Failed to send: $errorText',
          timestamp: DateTime.now(),
        ),
      );
      _emitMessages();
    }
  }

  @override
  Future<void> answerAskUser(
    String sessionId,
    String askUserId,
    String answer,
  ) async {
    // Find the ask-user message and update it optimistically.
    final index = _messages.indexWhere(
      (m) => m.askUser != null && m.askUser!.id == askUserId,
    );
    if (index == -1) return;

    final original = _messages[index];
    _messages[index] = original.copyWith(
      askUser: original.askUser!.copyWith(
        status: AskUserStatus.answered,
        answer: answer,
      ),
    );
    _emitMessages();

    final connection = _connectionManager.getConnection(_serverId);
    if (connection == null ||
        connection.status != WsConnectionStatus.connected) {
      _upsertMessage(
        ChatMessage(
          id: 'error-${DateTime.now().millisecondsSinceEpoch}',
          type: ChatMessageType.systemMessage,
          content: 'Failed to send reply: Server not connected',
          timestamp: DateTime.now(),
        ),
      );
      _emitMessages();
      return;
    }

    final result = await connection.client.sendAskUserReply(
      deviceToken: connection.server.deviceToken!,
      sessionId: _sessionId,
      question: original.askUser!.question,
      answer: answer,
    );

    if (!result.success) {
      final errorText = result.errorMessage.isEmpty
          ? 'Server not connected'
          : result.errorMessage;
      _upsertMessage(
        ChatMessage(
          id: 'error-${DateTime.now().millisecondsSinceEpoch}',
          type: ChatMessageType.systemMessage,
          content: 'Failed to send reply: $errorText',
          timestamp: DateTime.now(),
        ),
      );
      _emitMessages();
    }
  }

  @override
  Future<void> cancel(String sessionId) async {
    final connection = _connectionManager.getConnection(_serverId);
    if (connection == null ||
        connection.status != WsConnectionStatus.connected) {
      return;
    }

    await connection.client.cancelSession(
      deviceToken: connection.server.deviceToken!,
      sessionId: _sessionId,
    );

    // Optimistically reset processing state so Stop button disappears
    // immediately. Server will confirm via SessionStatusPayload later.
    if (_isProcessing) {
      _isProcessing = false;
      _processingController.add(false);
    }
  }

  @override
  Stream<List<ChatMessage>> watchMessages() => _messageController.stream;

  @override
  Stream<List<AgentInfo>>? watchAgents() => _agentController.stream;

  @override
  Stream<bool> watchProcessing() => _processingController.stream;

  // ---------------------------------------------------------------------------
  // Subscription
  // ---------------------------------------------------------------------------

  /// Subscribes to session events from the server.
  ///
  /// Loads persisted messages from the local database before subscribing,
  /// then registers a listener on [WsConnectionManager] to automatically
  /// re-subscribe whenever the connection becomes available again after
  /// a disconnect or reconnect.
  Future<void> subscribe() async {
    await _loadFromDb();
    _subscribeInternal();

    _connectionListener = _onConnectionChange;
    _connectionManager.addListener(_connectionListener!);
  }

  void _onConnectionChange() {
    if (_disposed) return;
    final conn = _connectionManager.getConnection(_serverId);
    final currentStatus = conn?.status ?? WsConnectionStatus.disconnected;

    // Only act on actual status transitions to avoid duplicate subscribes.
    if (currentStatus == _lastConnectionStatus) return;
    final previousStatus = _lastConnectionStatus;
    _lastConnectionStatus = currentStatus;

    if (currentStatus != WsConnectionStatus.connected) return;

    // Only re-subscribe if we don't have an active subscription.
    // WsBridgeClient handles proactive reconnect internally via
    // _resubscribeAll() — we must NOT create a new subscribe call here
    // because that overwrites the correct last_event_id with null.
    if (_subscription == null &&
        previousStatus != WsConnectionStatus.connected) {
      debugPrint('[WsChatRepository] Connection restored — re-subscribing');
      _subscribeInternal();
    }
  }

  void _subscribeInternal() {
    final connection = _connectionManager.getConnection(_serverId);
    if (connection == null ||
        connection.status != WsConnectionStatus.connected) {
      debugPrint(
        '[WsChatRepository] subscribe() SKIPPED: '
        'connection=${connection?.status} '
        'serverId=$_serverId',
      );
      return;
    }

    _subscription?.cancel();
    _subscription = null;

    debugPrint(
      '[WsChatRepository] subscribe() sessionId=$_sessionId '
      'serverId=$_serverId lastEventId=$_lastEventId',
    );

    final stream = connection.client.subscribeSession(
      deviceToken: connection.server.deviceToken!,
      sessionId: _sessionId,
      lastEventId: _lastEventId,
    );

    _subscription = stream.listen(
      _handleEvent,
      onError: (error) {
        debugPrint('[WsChatRepository] Subscription error: $error');
        _subscription?.cancel();
        _subscription = null;
      },
      onDone: () {
        // Stream ended (WsBridgeClient was disposed on reconnect).
        // Mark as unsubscribed so _onConnectionChange can re-subscribe.
        _subscription = null;
      },
      cancelOnError: false,
    );
  }

  /// Cleans up stream controllers and subscriptions.
  void dispose() {
    if (_disposed) return;
    _disposed = true;

    if (_connectionListener != null) {
      _connectionManager.removeListener(_connectionListener!);
      _connectionListener = null;
    }
    _subscription?.cancel();
    _subscription = null;
    _emitDebounce?.cancel();
    _emitDebounce = null;
    _seenEventIds.clear();
    _messageController.close();
    _processingController.close();
    _agentController.close();
  }

  // ---------------------------------------------------------------------------
  // Event handling
  // ---------------------------------------------------------------------------

  void _handleEvent(SessionEvent event) {
    // Deduplicate backfill events after proactive reconnect.
    final eventId = event.eventId;
    if (eventId.isNotEmpty && _seenEventIds.contains(eventId)) return;
    if (eventId.isNotEmpty) {
      _seenEventIds.add(eventId);
      if (_seenEventIds.length > _maxSeenIds) {
        _seenEventIds.remove(_seenEventIds.first);
      }
    }

    _lastEventId = eventId;

    final payload = event.payload;
    if (payload == null) return;

    switch (payload) {
      case UserMessagePayload():
        _handleUserMessage(event, payload);
      case AgentMessagePayload():
        _handleAgentMessage(event, payload);
      case ToolCallStartPayload():
        _handleToolCallStart(event, payload);
      case ToolCallEndPayload():
        _handleToolCallEnd(payload);
      case ReasoningPayload():
        _handleReasoning(event, payload);
      case AskUserPayload():
        _handleAskUser(event, payload);
      case PlanPayload():
        _handlePlan(event, payload);
      case SessionStatusPayload():
        _handleSessionStatus(event, payload);
      case ErrorPayload():
        _handleError(event, payload);
      case AgentLifecyclePayload():
        _handleAgentLifecycle(event, payload);
    }
  }

  void _handleUserMessage(SessionEvent event, UserMessagePayload payload) {
    // Remove the first optimistic user message (id starts with 'user-') that
    // has the same content. Only remove ONE to handle duplicate content correctly.
    final idx = _messages.indexWhere(
      (m) =>
          m.type == ChatMessageType.userMessage &&
          m.id.startsWith('user-') &&
          m.content == payload.content,
    );
    if (idx != -1) {
      _messages.removeAt(idx);
    }

    _upsertMessage(
      ChatMessage(
        id: event.eventId,
        type: ChatMessageType.userMessage,
        content: payload.content,
        timestamp: event.timestamp,
      ),
    );
    _emitMessages();
  }

  void _handleAgentMessage(SessionEvent event, AgentMessagePayload payload) {
    if (payload.isComplete) {
      // Remove streaming accumulator messages for this agent.
      // Step may differ between streaming (step N) and complete (step N+1),
      // so match by agentId prefix pattern instead of exact streamId.
      final agentPrefix = '${event.agentId}-';
      _messages.removeWhere(
        (m) =>
            m.type == ChatMessageType.agentMessage &&
            m.id.startsWith(agentPrefix),
      );

      _upsertMessage(
        ChatMessage(
          id: event.eventId,
          type: ChatMessageType.agentMessage,
          content: payload.content,
          timestamp: event.timestamp,
          agentId: event.agentId,
        ),
      );
      _emitMessages();
      return;
    }

    // Streaming: accumulate content using agentId-step as the key.
    final streamId = '${event.agentId}-${event.step}';
    final existingIndex = _messages.indexWhere((m) => m.id == streamId);

    if (existingIndex != -1) {
      final existing = _messages[existingIndex];
      _messages[existingIndex] = existing.copyWith(
        content: '${existing.content}${payload.content}',
      );
    } else {
      _messages.add(
        ChatMessage(
          id: streamId,
          type: ChatMessageType.agentMessage,
          content: payload.content,
          timestamp: event.timestamp,
          agentId: event.agentId,
        ),
      );
    }

    _emitMessages();
  }

  void _handleToolCallStart(SessionEvent event, ToolCallStartPayload payload) {
    _upsertMessage(
      ChatMessage(
        id: payload.callId,
        type: ChatMessageType.toolCall,
        content: '',
        timestamp: event.timestamp,
        agentId: event.agentId,
        toolCall: ToolCallData(
          id: payload.callId,
          toolName: payload.toolName,
          arguments: payload.arguments,
          status: ToolCallStatus.running,
        ),
      ),
    );
    _emitMessages();
  }

  void _handleToolCallEnd(ToolCallEndPayload payload) {
    final index = _messages.indexWhere((m) => m.id == payload.callId);
    if (index == -1) return;

    final existing = _messages[index];
    _messages[index] = existing.copyWith(
      toolCall: existing.toolCall!.copyWith(
        status: payload.hasError
            ? ToolCallStatus.failed
            : ToolCallStatus.completed,
        result: payload.hasError ? null : payload.resultSummary,
        error: payload.hasError ? payload.resultSummary : null,
      ),
    );
    _emitMessages();
  }

  void _handleReasoning(SessionEvent event, ReasoningPayload payload) {
    if (!payload.isComplete) return;

    _upsertMessage(
      ChatMessage(
        id: event.eventId,
        type: ChatMessageType.reasoning,
        content: payload.content,
        timestamp: event.timestamp,
        agentId: event.agentId,
      ),
    );
    _emitMessages();
  }

  void _handleAskUser(SessionEvent event, AskUserPayload payload) {
    if (payload.isAnswered) return;

    final askId = 'ask-${event.eventId}';
    _upsertMessage(
      ChatMessage(
        id: askId,
        type: ChatMessageType.askUser,
        content: '',
        timestamp: event.timestamp,
        agentId: event.agentId,
        askUser: AskUserData(
          id: askId,
          question: payload.question,
          options: payload.options,
          status: AskUserStatus.pending,
        ),
      ),
    );
    _emitMessages();
  }

  void _handlePlan(SessionEvent event, PlanPayload payload) {
    final planId = 'plan-${event.agentId}';
    _upsertMessage(
      ChatMessage(
        id: planId,
        type: ChatMessageType.planUpdate,
        content: '',
        timestamp: event.timestamp,
        agentId: event.agentId,
        plan: PlanData(
          goal: payload.planName,
          steps: [
            for (var i = 0; i < payload.steps.length; i++)
              PlanStep(
                index: i,
                description: payload.steps[i].title,
                status: _mapPlanStepStatus(payload.steps[i].status),
              ),
          ],
        ),
      ),
    );
    _emitMessages();
  }

  void _handleSessionStatus(SessionEvent event, SessionStatusPayload payload) {
    // ProcessingStarted confirms the CLI received the task — remove any
    // stale send-timeout error that was shown while the connection was dead.
    if (payload.state == MobileSessionState.active) {
      _messages.removeWhere((m) => m.id == _sendErrorId);
      _emitMessages();
    }

    // Track processing state: active = processing, anything else = idle.
    final nowProcessing = payload.state == MobileSessionState.active;
    if (nowProcessing != _isProcessing) {
      _isProcessing = nowProcessing;
      // Don't emit processing state during history backfill to avoid
      // stop→send button flickering on chat open.
      if (!_disposed && _historyLoaded) {
        _processingController.add(_isProcessing);
      }
    }

    // Status transitions (active/idle) are NOT shown as chat messages —
    // they only drive the stop/send button toggle via isProcessing.
    // Only show explicit server messages (e.g. error descriptions).
    if (payload.message.isNotEmpty) {
      _upsertMessage(
        ChatMessage(
          id: event.eventId,
          type: ChatMessageType.systemMessage,
          content: payload.message,
          timestamp: event.timestamp,
        ),
      );
      _emitMessages();
    }
  }

  void _handleError(SessionEvent event, ErrorPayload payload) {
    _upsertMessage(
      ChatMessage(
        id: event.eventId,
        type: ChatMessageType.systemMessage,
        content: 'Error [${payload.code}]: ${payload.message}',
        timestamp: event.timestamp,
      ),
    );
    _emitMessages();
  }

  void _handleAgentLifecycle(
    SessionEvent event,
    AgentLifecyclePayload payload,
  ) {
    final status = switch (payload.lifecycleType) {
      'agent_spawned' => AgentStatus.running,
      'agent_completed' => AgentStatus.completed,
      'agent_failed' => AgentStatus.failed,
      _ => AgentStatus.running,
    };

    final agent = AgentInfo(
      agentId: payload.agentId,
      description: payload.description,
      status: status,
      lastActivityAt: event.timestamp,
    );

    final existingIndex = _agents.indexWhere(
      (a) => a.agentId == payload.agentId,
    );
    if (existingIndex != -1) {
      _agents[existingIndex] = agent;
    } else {
      _agents.add(agent);
    }

    if (!_disposed) {
      _agentController.add(List.unmodifiable(_agents));
    }

    // Add lifecycle message to chat for visibility.
    _upsertMessage(
      ChatMessage(
        id: 'lifecycle-${event.eventId}',
        type: ChatMessageType.systemMessage,
        content: _formatLifecycleMessage(payload),
        timestamp: event.timestamp,
        agentId: payload.agentId,
      ),
    );
    _emitMessages();
  }

  String _formatLifecycleMessage(AgentLifecyclePayload payload) {
    final shortId = payload.agentId.replaceFirst('code-agent-', '');
    final label = payload.agentId == 'supervisor'
        ? 'Supervisor'
        : 'Code Agent [$shortId]';
    return switch (payload.lifecycleType) {
      'agent_spawned' => '+ $label spawned: "${payload.description}"',
      'agent_completed' => '✓ $label completed',
      'agent_failed' => '✗ $label failed',
      _ => '[${payload.lifecycleType}] $label',
    };
  }

  // ---------------------------------------------------------------------------
  // Connection helpers
  // ---------------------------------------------------------------------------

  /// Waits for the connection to become available (up to 15 seconds).
  /// Used when a reconnect is in progress at the time the user sends a message.
  Future<WsServerConnection?> _waitForConnection() async {
    final completer = Completer<WsServerConnection?>();

    void listener() {
      final conn = _connectionManager.getConnection(_serverId);
      if (conn != null && conn.status == WsConnectionStatus.connected) {
        if (!completer.isCompleted) completer.complete(conn);
      }
    }

    _connectionManager.addListener(listener);

    // Check immediately in case it reconnected between our check and adding
    // the listener (race condition).
    final conn = _connectionManager.getConnection(_serverId);
    if (conn != null && conn.status == WsConnectionStatus.connected) {
      _connectionManager.removeListener(listener);
      return conn;
    }

    try {
      return await completer.future.timeout(const Duration(seconds: 15));
    } on TimeoutException {
      return null;
    } finally {
      _connectionManager.removeListener(listener);
    }
  }

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  /// Inserts or replaces a message with the same [ChatMessage.id].
  void _upsertMessage(ChatMessage message) {
    final index = _messages.indexWhere((m) => m.id == message.id);
    if (index != -1) {
      _messages[index] = message;
    } else {
      _messages.add(message);
    }
  }

  /// Schedules a debounced emission of the current message list.
  ///
  /// Backfill events arrive one by one over ~100-300ms. Without debounce,
  /// each event triggers a UI rebuild showing messages appearing one at a time.
  /// The debounce collects all events within a short window and emits them
  /// as a single batch, giving a smooth "history loaded" appearance.
  ///
  /// For user-initiated actions (send, cancel) we want immediate feedback,
  /// but those go through optimistic updates that don't call _emitMessages.
  /// Emits messages with debounce during backfill, immediately during live.
  void _emitMessages() {
    if (_disposed) return;
    if (_historyLoaded) {
      _emitDebounce?.cancel();
      _flushMessages();
    } else {
      _emitDebounce?.cancel();
      _emitDebounce = Timer(const Duration(milliseconds: 150), _flushMessages);
    }
  }

  void _flushMessages() {
    if (_disposed) return;

    final wasFirstBatch = !_historyLoaded;
    _historyLoaded = true;

    // Emit the final processing state after backfill completes so the UI
    // shows the correct button (send vs stop) without intermediate flicker.
    if (wasFirstBatch && _isProcessing) {
      _processingController.add(true);
    }

    // Sort by timestamp so replayed events (from re-subscribe after reconnect)
    // appear in correct chronological order.
    _messages.sort((a, b) => a.timestamp.compareTo(b.timestamp));

    // Trim oldest messages to keep UI performant.
    if (_messages.length > _maxMessages) {
      _messages.removeRange(0, _messages.length - _maxMessages);
    }

    _messageController.add(List.unmodifiable(_messages));

    // Persist to local database (fire-and-forget).
    _persistMessages();
  }

  /// Writes the current message list to SQLite.
  ///
  /// Runs asynchronously without blocking the UI. Errors are logged but
  /// do not interrupt the message stream.
  void _persistMessages() {
    if (_messageStore == null) return;

    // Filter out transient streaming messages (id pattern: "agentId-step").
    // Only persist messages with stable IDs.
    final persistable = _messages.where(_isPersistable).toList();
    if (persistable.isEmpty) return;

    _messageStore.upsertMessages(_sessionId, persistable).catchError((e) {
      debugPrint('[WsChatRepository] Failed to persist messages: $e');
    });
  }

  /// Returns true if a message should be persisted to the database.
  ///
  /// Excludes transient streaming accumulator messages (which have synthetic
  /// IDs like "agentId-step") and temporary optimistic user messages (which
  /// have IDs like "user-timestamp"). These are replaced by server-confirmed
  /// versions with real event IDs.
  static bool _isPersistable(ChatMessage message) {
    // Optimistic user messages are replaced by server-confirmed ones.
    if (message.id.startsWith('user-')) return false;
    // Send-timeout error is transient.
    if (message.id == _sendErrorId) return false;
    // Streaming accumulator messages have synthetic IDs like "agentId-step".
    // Real event IDs from the server are UUIDs or longer alphanumeric strings.
    // A simple heuristic: if the ID contains a hyphen and the part after the
    // last hyphen is a pure number, it's likely a streaming accumulator.
    final lastHyphen = message.id.lastIndexOf('-');
    if (lastHyphen != -1) {
      final suffix = message.id.substring(lastHyphen + 1);
      if (suffix.isNotEmpty && int.tryParse(suffix) != null) {
        // But allow IDs like "ask-uuid" and "plan-agentId" and "error-timestamp"
        // and "lifecycle-uuid". Only skip if prefix looks like an agent ID.
        final prefix = message.id.substring(0, lastHyphen);
        if (prefix.startsWith('code-agent') || prefix == 'supervisor') {
          return false;
        }
      }
    }
    return true;
  }

  static PlanStepStatus _mapPlanStepStatus(WsPlanStepStatus status) {
    return switch (status) {
      WsPlanStepStatus.completed => PlanStepStatus.completed,
      WsPlanStepStatus.inProgress => PlanStepStatus.inProgress,
      WsPlanStepStatus.pending => PlanStepStatus.pending,
      WsPlanStepStatus.failed => PlanStepStatus.failed,
      WsPlanStepStatus.unspecified => PlanStepStatus.pending,
    };
  }
}

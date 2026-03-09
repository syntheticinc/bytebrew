import 'dart:async';

import 'package:flutter/foundation.dart';

import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/utils/debug_file_logger.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';

/// [ChatRepository] implementation backed by WebSocket via [WsConnectionManager].
///
/// Manages an internal message list, handles optimistic updates for user
/// actions, and processes session events from the server subscription.
class WsChatRepository implements ChatRepository {
  WsChatRepository({
    required WsConnectionManager connectionManager,
    required String serverId,
    required String sessionId,
  }) : _connectionManager = connectionManager,
       _serverId = serverId,
       _sessionId = sessionId;

  final WsConnectionManager _connectionManager;
  final String _serverId;
  final String _sessionId;

  final List<ChatMessage> _messages = [];
  final StreamController<List<ChatMessage>> _messageController =
      StreamController<List<ChatMessage>>.broadcast();

  StreamSubscription<SessionEvent>? _subscription;
  String? _lastEventId;
  bool _disposed = false;
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

  @override
  Future<List<ChatMessage>> getMessages(String sessionId) async {
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
      dlog('[WsChatRepository] Connection not ready (${connection.status}), waiting for reconnect...');
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

    dlog('[WsChatRepository] sendNewTask: sessionId=$_sessionId text=$text');
    final result = await connection.client.sendNewTask(
      deviceToken: connection.server.deviceToken!,
      sessionId: _sessionId,
      task: text,
    );
    dlog(
      '[WsChatRepository] sendNewTask result: success=${result.success} '
      'error=${result.errorMessage}',
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
          id: isTimeout ? _sendErrorId : 'error-${DateTime.now().millisecondsSinceEpoch}',
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
  }

  @override
  Stream<List<ChatMessage>> watchMessages() => _messageController.stream;

  @override
  Stream<List<AgentInfo>>? watchAgents() => null;

  // ---------------------------------------------------------------------------
  // Subscription
  // ---------------------------------------------------------------------------

  /// Subscribes to session events from the server.
  ///
  /// Registers a listener on [WsConnectionManager] to automatically
  /// re-subscribe whenever the connection becomes available again after
  /// a disconnect or reconnect.
  void subscribe() {
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

    if (currentStatus != WsConnectionStatus.connected) {
      _subscription?.cancel();
      _subscription = null;
      return;
    }

    // Re-subscribe only on transition TO connected (from non-connected).
    if (previousStatus != WsConnectionStatus.connected) {
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
    _messageController.close();
  }

  // ---------------------------------------------------------------------------
  // Event handling
  // ---------------------------------------------------------------------------

  void _handleEvent(SessionEvent event) {
    _lastEventId = event.eventId;

    final payload = event.payload;
    if (payload == null) {
      dlog(
        '[WsChatRepository] Event type=${event.type.name} '
        'has null payload → skipped',
      );
      return;
    }

    dlog(
      '[WsChatRepository] Handling event: type=${event.type.name} '
      'payload=${payload.runtimeType}',
    );

    switch (payload) {
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
    }
  }

  void _handleAgentMessage(SessionEvent event, AgentMessagePayload payload) {
    dlog(
      '[WsChatRepository] AgentMessage: isComplete=${payload.isComplete} '
      'content="${payload.content.length > 80 ? '${payload.content.substring(0, 80)}...' : payload.content}"',
    );
    if (payload.isComplete) {
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
    }

    final content = payload.message.isNotEmpty
        ? payload.message
        : 'Session status: ${payload.state.name}';

    _upsertMessage(
      ChatMessage(
        id: event.eventId,
        type: ChatMessageType.systemMessage,
        content: content,
        timestamp: event.timestamp,
      ),
    );
    _emitMessages();
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

  void _emitMessages() {
    if (_disposed) return;
    // Sort by timestamp so replayed events (from re-subscribe after reconnect)
    // appear in correct chronological order.
    _messages.sort((a, b) => a.timestamp.compareTo(b.timestamp));
    _messageController.add(List.unmodifiable(_messages));
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

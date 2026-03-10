import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import 'package:bytebrew_mobile/core/crypto/message_cipher.dart';
import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';

/// High-level client for CLI communication via Bridge WebSocket.
///
/// Handles request-response matching via `request_id`, encryption of
/// payloads when a shared secret is available, and event stream parsing.
class WsBridgeClient {
  WsBridgeClient({
    required WsConnection connection,
    this.cipher,
    this.deviceId = '',
  }) : _connection = connection {
    _messageSubscription = _connection.messages.listen(_onMessage);
  }

  final WsConnection _connection;
  final MessageCipher? cipher;
  final String deviceId;

  StreamSubscription<Map<String, dynamic>>? _messageSubscription;

  /// Pending request completers, keyed by request_id.
  final Map<String, Completer<Map<String, dynamic>>> _pendingRequests = {};

  /// Controllers for event streams (e.g. session_event subscriptions).
  final Map<String, StreamController<Map<String, dynamic>>> _eventControllers =
      {};

  int _requestCounter = 0;
  int _sendCounter = 0;
  bool _disposed = false;

  // -------------------------------------------------------------------------
  // Public API
  // -------------------------------------------------------------------------

  /// Pings the CLI server.
  Future<PingResult> ping() async {
    final response = await _sendRequest('ping', {});
    final payload = response['payload'] as Map<String, dynamic>? ?? {};
    return PingResult(
      timestamp: DateTime.now(),
      serverName: payload['server_name'] as String? ?? '',
      serverId: payload['server_id'] as String? ?? '',
    );
  }

  /// Pairs this device with the CLI server.
  Future<PairResult> pair({
    required String token,
    required String deviceName,
    Uint8List? mobilePublicKey,
  }) async {
    final payload = <String, dynamic>{
      'token': token,
      'device_name': deviceName,
    };
    if (mobilePublicKey != null) {
      payload['device_public_key'] = base64Encode(mobilePublicKey);
    }

    // Pairing is always plaintext (before shared secret is established).
    final response = await _sendRequest(
      'pair_request',
      payload,
      encrypt: false,
    );

    // Check for error response from CLI.
    if (response['type'] == 'error') {
      final errPayload = response['payload'] as Map<String, dynamic>? ?? {};
      final msg = errPayload['message'] as String? ?? 'Pairing failed';
      throw Exception(msg);
    }

    final respPayload = response['payload'] as Map<String, dynamic>? ?? {};

    Uint8List? serverPublicKey;
    final serverPubKeyStr = respPayload['server_public_key'] as String?;
    if (serverPubKeyStr != null && serverPubKeyStr.isNotEmpty) {
      serverPublicKey = base64Decode(serverPubKeyStr);
    }

    return PairResult(
      deviceId: respPayload['device_id'] as String? ?? '',
      deviceToken: respPayload['device_token'] as String? ?? '',
      serverName: respPayload['server_name'] as String? ?? '',
      serverId: respPayload['server_id'] as String? ?? '',
      serverPublicKey: serverPublicKey,
    );
  }

  /// Lists all sessions on the CLI server.
  Future<ListSessionsResult> listSessions({required String deviceToken}) async {
    final response = await _sendRequest('list_sessions', {
      'device_token': deviceToken,
    });

    final payload = response['payload'] as Map<String, dynamic>? ?? {};
    final sessionsJson = payload['sessions'] as List<dynamic>? ?? [];

    final sessions = sessionsJson.map((json) {
      final s = json as Map<String, dynamic>;
      return MobileSession(
        sessionId: s['session_id'] as String? ?? '',
        projectKey: s['project_name'] as String? ?? s['project_key'] as String? ?? '',
        projectRoot: s['project_root'] as String? ?? '',
        status: _parseSessionState(s['status'] as String?),
        currentTask: s['current_task'] as String? ?? '',
        startedAt: _parseDateTime(s['started_at']),
        lastActivityAt: _parseDateTime(s['last_activity_at']),
        hasAskUser: s['has_ask_user'] as bool? ?? false,
        platform: s['platform'] as String? ?? '',
      );
    }).toList();

    return ListSessionsResult(
      sessions: sessions,
      serverName: payload['server_name'] as String? ?? '',
      serverId: payload['server_id'] as String? ?? '',
    );
  }

  /// Subscribes to events for a specific session.
  ///
  /// Returns a stream of parsed [SessionEvent] objects.
  Stream<SessionEvent> subscribeSession({
    required String deviceToken,
    required String sessionId,
    String? lastEventId,
  }) {
    final controller = StreamController<SessionEvent>();
    final eventKey = 'subscribe-$sessionId';

    // Create a raw event controller for this subscription.
    final rawController = StreamController<Map<String, dynamic>>.broadcast();
    _eventControllers[eventKey] = rawController;

    // Parse raw events into SessionEvent objects.
    rawController.stream.listen(
      (json) {
        final event = _parseSessionEvent(json);
        if (event != null) {
          controller.add(event);
        }
      },
      onError: controller.addError,
      onDone: controller.close,
    );

    controller.onCancel = () {
      rawController.close();
      // Guard against async race: onCancel fires as a microtask, so by the
      // time it runs, _subscribeInternal may have already registered a NEW
      // controller for the same key. Only remove if still ours.
      if (_eventControllers[eventKey] == rawController) {
        _eventControllers.remove(eventKey);
      }
    };

    // Send the subscribe request (fire and forget).
    final payload = <String, dynamic>{
      'device_token': deviceToken,
      'session_id': sessionId,
    };
    if (lastEventId != null) {
      payload['last_event_id'] = lastEventId;
    }

    _sendRequestFireAndForget('subscribe', payload);

    return controller.stream;
  }

  /// Sends a new task to the CLI server.
  Future<SendCommandResult> sendNewTask({
    required String deviceToken,
    required String sessionId,
    required String task,
  }) async {
    return _sendCommand('new_task', {
      'device_token': deviceToken,
      'session_id': sessionId,
      'content': task,
    });
  }

  /// Sends a reply to an ask-user prompt.
  Future<SendCommandResult> sendAskUserReply({
    required String deviceToken,
    required String sessionId,
    required String question,
    required String answer,
  }) async {
    return _sendCommand('ask_user_reply', {
      'device_token': deviceToken,
      'session_id': sessionId,
      'question': question,
      'answer': answer,
    });
  }

  /// Cancels an active session.
  Future<SendCommandResult> cancelSession({
    required String deviceToken,
    required String sessionId,
  }) async {
    return _sendCommand('cancel_session', {
      'device_token': deviceToken,
      'session_id': sessionId,
    });
  }

  /// Disposes the client. Does NOT dispose the underlying [WsConnection].
  Future<void> dispose() async {
    if (_disposed) return;
    _disposed = true;

    await _messageSubscription?.cancel();
    _messageSubscription = null;

    for (final completer in _pendingRequests.values) {
      if (!completer.isCompleted) {
        completer.completeError(
          StateError('WsBridgeClient disposed while request pending'),
        );
      }
    }
    _pendingRequests.clear();

    for (final controller in _eventControllers.values) {
      await controller.close();
    }
    _eventControllers.clear();
  }

  // -------------------------------------------------------------------------
  // Internal: request/response
  // -------------------------------------------------------------------------

  Future<Map<String, dynamic>> _sendRequest(
    String type,
    Map<String, dynamic> payload, {
    bool encrypt = true,
  }) async {
    // If the connection is stale (e.g. after Android idle/doze suspended
    // timers and the TCP connection died silently), wait for reconnect
    // before sending. The ping timer's stale check or the pong watchdog
    // will trigger the reconnect.
    if (_connection.isStale) {
      await _waitForReconnect();
    }

    final requestId = _generateRequestId();
    final completer = Completer<Map<String, dynamic>>();
    _pendingRequests[requestId] = completer;

    final message = await _buildMessage(
      type,
      requestId,
      payload,
      encrypt: encrypt,
    );
    _connection.send(message);

    // Timeout after 30 seconds.
    return completer.future.timeout(
      const Duration(seconds: 30),
      onTimeout: () {
        _pendingRequests.remove(requestId);
        throw TimeoutException(
          'Request $type timed out',
          const Duration(seconds: 30),
        );
      },
    );
  }

  Future<void> _sendRequestFireAndForget(
    String type,
    Map<String, dynamic> payload,
  ) async {
    final requestId = _generateRequestId();
    final message = await _buildMessage(type, requestId, payload);
    _connection.send(message);
  }

  Future<SendCommandResult> _sendCommand(
    String type,
    Map<String, dynamic> payload,
  ) async {
    // Verify connection is alive before sending important commands.
    // This catches silently-dead TCP connections (common on Android after
    // idle) before wasting 30s on a timeout.
    if (!await _connection.ensureAlive()) {
      try {
        await _waitForReconnect();
      } on TimeoutException {
        return SendCommandResult(
          success: false,
          errorMessage: 'Connection lost — please try again',
        );
      }
    }

    try {
      final response = await _sendRequest(type, payload);
      final respPayload = response['payload'] as Map<String, dynamic>? ?? {};
      final error = respPayload['error'] as String? ?? '';
      return SendCommandResult(success: error.isEmpty, errorMessage: error);
    } on Exception catch (e) {
      return SendCommandResult(success: false, errorMessage: e.toString());
    }
  }

  /// Waits for the underlying [WsConnection] to transition back to
  /// [WsConnectionStatus.connected] after a stale-connection reconnect.
  Future<void> _waitForReconnect() async {
    final completer = Completer<void>();

    late final StreamSubscription<WsConnectionStatus> sub;
    sub = _connection.statusChanges.listen((status) {
      if (status == WsConnectionStatus.connected) {
        sub.cancel();
        if (!completer.isCompleted) completer.complete();
      }
    });

    // If the connection is already reconnected by the time we subscribe
    // (race condition), complete immediately.
    if (!_connection.isStale &&
        _connection.status == WsConnectionStatus.connected) {
      sub.cancel();
      if (!completer.isCompleted) completer.complete();
      return;
    }

    try {
      await completer.future.timeout(const Duration(seconds: 15));
    } on TimeoutException {
      sub.cancel();
      throw TimeoutException(
        'Connection reconnect timed out',
        const Duration(seconds: 15),
      );
    }
  }

  Future<Map<String, dynamic>> _buildMessage(
    String type,
    String requestId,
    Map<String, dynamic> payload, {
    bool encrypt = true,
  }) async {
    // Build the inner (CLI-level) message.
    final innerMessage = <String, dynamic>{
      'type': type,
      'request_id': requestId,
      'device_id': deviceId,
      'payload': payload,
    };

    if (encrypt && cipher != null) {
      // Encrypt the entire inner message to a base64 string.
      final jsonBytes = utf8.encode(jsonEncode(innerMessage));
      final encrypted = await cipher!.encrypt(
        Uint8List.fromList(jsonBytes),
        _sendCounter++,
      );
      return <String, dynamic>{
        'type': 'data',
        'payload': base64Encode(encrypted),
      };
    }

    return <String, dynamic>{'type': 'data', 'payload': innerMessage};
  }

  // -------------------------------------------------------------------------
  // Internal: message handling
  // -------------------------------------------------------------------------

  void _onMessage(Map<String, dynamic> bridgeMessage) {
    final bridgeType = bridgeMessage['type'] as String?;
    if (bridgeType != 'data') {
      debugPrint('[WsBridgeClient] Ignoring bridge message type=$bridgeType');
      return;
    }

    final rawPayload = bridgeMessage['payload'];
    if (rawPayload == null) {
      debugPrint('[WsBridgeClient] Ignoring data message with null payload');
      return;
    }

    Map<String, dynamic> innerMessage;

    if (rawPayload is String) {
      // Encrypted payload (base64) -- decrypt.
      if (cipher != null) {
        _decryptAndHandle(rawPayload);
        return;
      }
      // Try parsing as JSON string.
      try {
        innerMessage = jsonDecode(rawPayload) as Map<String, dynamic>;
      } on FormatException {
        debugPrint('[WsBridgeClient] Cannot parse payload string');
        return;
      }
    } else if (rawPayload is Map<String, dynamic>) {
      innerMessage = rawPayload;
    } else {
      debugPrint(
        '[WsBridgeClient] Unexpected payload type: '
        '${rawPayload.runtimeType}',
      );
      return;
    }

    _handleInnerMessage(innerMessage);
  }

  Future<void> _decryptAndHandle(String base64Payload) async {
    try {
      final encrypted = base64Decode(base64Payload);
      final (decrypted, _) = await cipher!.decrypt(encrypted);
      final json = jsonDecode(utf8.decode(decrypted)) as Map<String, dynamic>;
      _handleInnerMessage(json);
    } on Exception catch (e) {
      debugPrint('[WsBridgeClient] Decrypt failed: $e');
    }
  }

  void _handleInnerMessage(Map<String, dynamic> message) {
    final type = message['type'] as String?;
    final requestId = message['request_id'] as String?;

    // Check if this is a response to a pending request.
    if (requestId != null && _pendingRequests.containsKey(requestId)) {
      final completer = _pendingRequests.remove(requestId);
      completer?.complete(message);
      return;
    }

    // Check if this is a session event.
    if (type == 'session_event') {
      _handleSessionEvent(message);
      return;
    }

    // Check for response types that match pending requests by type.
    if (type != null && requestId != null) {
      // Try to find a matching pending request.
      final completer = _pendingRequests.remove(requestId);
      if (completer != null) {
        completer.complete(message);
      }
    }
  }

  void _handleSessionEvent(Map<String, dynamic> message) {
    final payload = message['payload'] as Map<String, dynamic>?;
    if (payload == null) {
      debugPrint('[WsBridgeClient] session_event with null payload');
      return;
    }

    final sessionId = payload['session_id'] as String?;
    if (sessionId == null) {
      debugPrint('[WsBridgeClient] session_event missing session_id');
      return;
    }

    final eventKey = 'subscribe-$sessionId';
    final controller = _eventControllers[eventKey];
    if (controller != null) {
      controller.add(message);
    }
  }

  // -------------------------------------------------------------------------
  // Internal: event parsing
  // -------------------------------------------------------------------------

  SessionEvent? _parseSessionEvent(Map<String, dynamic> message) {
    final payload = message['payload'] as Map<String, dynamic>?;
    if (payload == null) return null;

    final sessionId = payload['session_id'] as String? ?? '';
    final eventJson = payload['event'] as Map<String, dynamic>?;
    if (eventJson == null) return null;

    final rawType = eventJson['type'] as String?;
    final eventRole = eventJson['role'] as String?;
    final eventType = _parseEventType(rawType, role: eventRole);
    final eventPayload = _parseEventPayload(eventType, eventJson);

    // event_id lives at the payload level, not inside the nested event object.
    final eventId = payload['event_id'] as String? ??
        'evt-${DateTime.now().millisecondsSinceEpoch}';

    return SessionEvent(
      eventId: eventId,
      sessionId: sessionId,
      type: eventType,
      timestamp: _parseDateTime(eventJson['timestamp']),
      agentId: eventJson['agent_id'] as String? ?? '',
      step: eventJson['step'] as int? ?? 0,
      payload: eventPayload,
    );
  }

  SessionEventType _parseEventType(String? type, {String? role}) {
    // Only show assistant messages from MessageCompleted events.
    // Skip user (mobile shows optimistically) and tool (handled via
    // ToolExecutionStarted/Completed events) messages.
    if (type == 'MessageCompleted' && role != null && role != 'assistant') {
      return SessionEventType.unspecified;
    }
    return switch (type) {
      'MessageChunk' ||
      'MessageCompleted' => SessionEventType.agentMessage,
      'ToolExecutionStarted' => SessionEventType.toolCallStart,
      'ToolExecutionCompleted' => SessionEventType.toolCallEnd,
      'ReasoningChunk' || 'ReasoningCompleted' => SessionEventType.reasoning,
      'AskUserRequested' || 'AskUserAnswered' => SessionEventType.askUser,
      'PlanUpdated' => SessionEventType.plan,
      'ProcessingStarted' ||
      'ProcessingCompleted' ||
      'ProcessingStopped' => SessionEventType.sessionStatus,
      // Progress-only events — silently ignored (no chat message).
      'MessageStarted' ||
      'StreamingProgress' ||
      'MessageChunkCompleted' ||
      'AgentLifecycle' => SessionEventType.unspecified,
      'Error' || 'ErrorOccurred' => SessionEventType.error,
      _ => SessionEventType.unspecified,
    };
  }

  SessionEventPayload? _parseEventPayload(
    SessionEventType type,
    Map<String, dynamic> eventJson,
  ) {
    return switch (type) {
      SessionEventType.agentMessage => AgentMessagePayload(
        content: eventJson['content'] as String? ?? '',
        isComplete: eventJson['type'] == 'MessageCompleted',
      ),
      SessionEventType.toolCallStart => ToolCallStartPayload(
        callId: eventJson['call_id'] as String? ?? '',
        toolName: eventJson['tool_name'] as String? ?? '',
        arguments: _parseStringMap(eventJson['arguments']),
      ),
      SessionEventType.toolCallEnd => ToolCallEndPayload(
        callId: eventJson['call_id'] as String? ?? '',
        toolName: eventJson['tool_name'] as String? ?? '',
        resultSummary: eventJson['result_summary'] as String? ?? '',
        hasError: eventJson['has_error'] as bool? ?? false,
      ),
      SessionEventType.reasoning => ReasoningPayload(
        content: eventJson['content'] as String? ?? '',
        isComplete: eventJson['type'] == 'ReasoningCompleted',
      ),
      SessionEventType.askUser => AskUserPayload(
        question: eventJson['question'] as String? ?? '',
        options: _parseStringList(eventJson['options']),
        isAnswered: eventJson['type'] == 'AskUserAnswered',
      ),
      SessionEventType.plan => PlanPayload(
        planName: eventJson['plan_name'] as String? ?? '',
        steps: _parsePlanSteps(eventJson['steps']),
      ),
      SessionEventType.sessionStatus => SessionStatusPayload(
        state: _parseSessionState(eventJson['state'] as String?),
        message: eventJson['message'] as String? ?? '',
      ),
      SessionEventType.error => ErrorPayload(
        code: eventJson['code'] as String? ?? '',
        message: eventJson['message'] as String? ?? '',
      ),
      SessionEventType.unspecified => null,
    };
  }

  // -------------------------------------------------------------------------
  // Internal: helpers
  // -------------------------------------------------------------------------

  String _generateRequestId() {
    final now = DateTime.now().millisecondsSinceEpoch;
    return 'req-$now-${_requestCounter++}';
  }

  static DateTime _parseDateTime(dynamic value) {
    if (value is String && value.isNotEmpty) {
      return DateTime.tryParse(value) ?? DateTime.now();
    }
    if (value is int) {
      return DateTime.fromMillisecondsSinceEpoch(value);
    }
    return DateTime.now();
  }

  static MobileSessionState _parseSessionState(String? state) {
    return switch (state) {
      'active' => MobileSessionState.active,
      'idle' => MobileSessionState.idle,
      'needs_attention' => MobileSessionState.needsAttention,
      'completed' => MobileSessionState.completed,
      'failed' => MobileSessionState.failed,
      'processing' => MobileSessionState.active,
      _ => MobileSessionState.unspecified,
    };
  }

  static Map<String, String> _parseStringMap(dynamic value) {
    if (value is! Map) return {};
    return value.map((key, val) => MapEntry(key.toString(), val.toString()));
  }

  static List<String> _parseStringList(dynamic value) {
    if (value is! List) return [];
    return value.map((e) => e.toString()).toList();
  }

  static List<PlanStepPayload> _parsePlanSteps(dynamic value) {
    if (value is! List) return [];
    return value.map((step) {
      final s = step as Map<String, dynamic>;
      return PlanStepPayload(
        title: s['title'] as String? ?? '',
        status: _parseWsPlanStepStatus(s['status'] as String?),
      );
    }).toList();
  }

  static WsPlanStepStatus _parseWsPlanStepStatus(String? status) {
    return switch (status) {
      'completed' => WsPlanStepStatus.completed,
      'in_progress' => WsPlanStepStatus.inProgress,
      'pending' => WsPlanStepStatus.pending,
      'failed' => WsPlanStepStatus.failed,
      _ => WsPlanStepStatus.unspecified,
    };
  }
}

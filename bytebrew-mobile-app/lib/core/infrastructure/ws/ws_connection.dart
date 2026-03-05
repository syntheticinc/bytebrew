import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';

part 'ws_connection.g.dart';

/// Connection status for a WebSocket link to the CLI proxy.
enum WsConnectionStatus { disconnected, connecting, connected, error }

/// WebSocket client that connects to the CLI MobileProxyServer.
///
/// Protocol:
/// - Incoming: `{"type":"init",...}`, `{"type":"event","event":{...}}`
/// - Outgoing: `{"type":"user_message",...}`, `{"type":"ask_user_answer",...}`,
///             `{"type":"cancel"}`
@Riverpod(keepAlive: true)
class WsConnection extends _$WsConnection {
  /// Maximum number of reconnect attempts before giving up.
  static const maxReconnectAttempts = 5;

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  StreamSubscription<bool>? _statusSubscription;

  // Init data received from the CLI.
  List<Map<String, dynamic>> _initMessages = [];
  Map<String, dynamic> _meta = {};

  // Event stream for downstream consumers.
  final _eventController = StreamController<Map<String, dynamic>>.broadcast();

  /// Stream of parsed event objects from the CLI.
  Stream<Map<String, dynamic>> get events => _eventController.stream;

  // Session processing status.
  bool _isProcessing = false;
  bool _hasAskUser = false;

  // Chat repository created on init.
  ChatRepository? _repository;
  WsChatRepository? _wsRepository;

  /// Initial message snapshots received in the `init` payload.
  List<Map<String, dynamic>> get initMessages => _initMessages;

  /// Metadata from the `init` payload (projectName, sessionId, etc.).
  Map<String, dynamic> get meta => _meta;

  /// Whether the agent is currently processing a request.
  bool get isProcessing => _isProcessing;

  /// Whether there is a pending ask-user prompt.
  bool get hasAskUser => _hasAskUser;

  /// Chat repository backed by this connection, or null if not connected.
  ChatRepository? get repository => _repository;

  /// Typed WS chat repository, for internal use and tests.
  @visibleForTesting
  WsChatRepository? get wsRepository => _wsRepository;

  /// Last error message, if any.
  String? get lastError => _lastError;
  String? _lastError;

  // Reconnect state.
  Timer? _reconnectTimer;
  int _reconnectAttempt = 0;
  String? _lastWsUrl;

  /// The URL last passed to [connect], for reconnection.
  @visibleForTesting
  set lastWsUrl(String? value) => _lastWsUrl = value;

  /// Current reconnect attempt count.
  @visibleForTesting
  int get reconnectAttempts => _reconnectAttempt;

  @visibleForTesting
  set reconnectAttempts(int value) => _reconnectAttempt = value;

  /// Factory for creating WebSocket channels. Override in tests.
  @visibleForTesting
  WebSocketChannel Function(Uri uri)? channelFactory;

  @override
  WsConnectionStatus build() => WsConnectionStatus.disconnected;

  /// Connects to the CLI MobileProxyServer at [wsUrl].
  ///
  /// [wsUrl] format: `ws://host:port`
  Future<void> connect(String wsUrl) async {
    // Clean up previous connection resources.
    _subscription?.cancel();
    _statusSubscription?.cancel();
    _channel?.sink.close();

    _lastWsUrl = wsUrl;
    _reconnectAttempt = 0;
    _reconnectTimer?.cancel();

    state = WsConnectionStatus.connecting;
    try {
      final uri = Uri.parse(wsUrl);
      _channel = channelFactory != null
          ? channelFactory!(uri)
          : WebSocketChannel.connect(uri);
      await _channel!.ready;
      state = WsConnectionStatus.connected;
      _lastError = null;
      _subscription = _channel!.stream.listen(
        _handleMessage,
        onError: (Object e) {
          _lastError = e.toString();
          state = WsConnectionStatus.error;
        },
        onDone: () {
          state = WsConnectionStatus.disconnected;
          scheduleReconnect();
        },
      );
    } on SocketException catch (e) {
      _lastError = 'Network error: ${e.message}';
      state = WsConnectionStatus.error;
    } catch (e) {
      _lastError = e.toString();
      state = WsConnectionStatus.error;
      scheduleReconnect();
    }
  }

  /// Disconnects from the CLI and stops reconnection attempts.
  void disconnect() {
    _reconnectTimer?.cancel();
    _reconnectAttempt = 0;
    _subscription?.cancel();
    _statusSubscription?.cancel();
    _channel?.sink.close();
    _channel = null;
    _repository = null;
    _wsRepository = null;
    _initMessages = [];
    _meta = {};
    _isProcessing = false;
    _hasAskUser = false;
    state = WsConnectionStatus.disconnected;
  }

  /// Sends a user message to the CLI.
  void sendUserMessage(String text) {
    _channel?.sink.add(jsonEncode({'type': 'user_message', 'text': text}));
  }

  /// Sends an ask-user answer to the CLI.
  void sendAskUserAnswer(String question, String answer) {
    _channel?.sink.add(
      jsonEncode({
        'type': 'ask_user_answer',
        'answers': [
          {'question': question, 'answer': answer},
        ],
      }),
    );
  }

  /// Sends a cancel request to the CLI.
  void sendCancel() {
    _channel?.sink.add(jsonEncode({'type': 'cancel'}));
  }

  void _handleMessage(dynamic raw) {
    final json = jsonDecode(raw as String) as Map<String, dynamic>;
    final type = json['type'] as String?;

    if (type == 'init') {
      _initMessages =
          (json['messages'] as List<dynamic>?)?.cast<Map<String, dynamic>>() ??
          [];
      _meta = json['meta'] as Map<String, dynamic>? ?? {};

      // Create the chat repository and load initial messages.
      final repo = WsChatRepository(connection: this);
      repo.loadInitMessages();
      _wsRepository = repo;
      _repository = repo;
      return;
    }

    if (type == 'event') {
      final event = json['event'] as Map<String, dynamic>?;
      if (event == null) return;
      final eventType = event['type'] as String?;

      if (eventType == 'heartbeat') return;
      if (eventType == 'ProcessingStarted') _isProcessing = true;
      if (eventType == 'ProcessingStopped') _isProcessing = false;
      if (eventType == 'AskUserRequested') _hasAskUser = true;
      if (eventType == 'AskUserResolved') _hasAskUser = false;

      _eventController.add(event);
    }
  }

  /// Schedules a reconnect attempt with exponential backoff.
  @visibleForTesting
  void scheduleReconnect() {
    if (_reconnectAttempt >= maxReconnectAttempts) {
      _lastError = 'Max reconnection attempts ($maxReconnectAttempts) reached';
      state = WsConnectionStatus.error;
      return;
    }
    final delay = Duration(seconds: 1 << _reconnectAttempt);
    _reconnectAttempt++;
    _reconnectTimer = Timer(delay, () {
      if (_lastWsUrl != null) connect(_lastWsUrl!);
    });
  }
}

/// Whether there is an active WebSocket connection.
@Riverpod(keepAlive: true)
bool hasActiveConnection(Ref ref) {
  final status = ref.watch(wsConnectionProvider);
  return status == WsConnectionStatus.connected;
}

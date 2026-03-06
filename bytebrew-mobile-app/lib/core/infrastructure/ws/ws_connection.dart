import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// Low-level WebSocket connection to the Bridge relay.
///
/// Connects to `ws://bridge/connect?server_id=xxx&device_id=yyy` and provides
/// typed JSON message send/receive. Handles reconnection with exponential
/// backoff.
class WsConnection {
  WsConnection({
    required this.bridgeUrl,
    required this.serverId,
    required this.deviceId,
  });

  final String bridgeUrl;
  final String serverId;
  final String deviceId;

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _channelSubscription;

  final _messageController = StreamController<Map<String, dynamic>>.broadcast();
  final _statusController = StreamController<WsConnectionStatus>.broadcast();

  WsConnectionStatus _status = WsConnectionStatus.disconnected;
  Timer? _reconnectTimer;
  int _reconnectAttempts = 0;
  bool _disposed = false;
  bool _intentionalClose = false;

  static const _maxReconnectDelay = 30;

  /// Current connection status.
  WsConnectionStatus get status => _status;

  /// Stream of parsed JSON messages from Bridge.
  Stream<Map<String, dynamic>> get messages => _messageController.stream;

  /// Stream of connection status changes.
  Stream<WsConnectionStatus> get statusChanges => _statusController.stream;

  /// Connects to Bridge.
  Future<void> connect() async {
    if (_disposed) return;
    if (_status == WsConnectionStatus.connected) return;

    _intentionalClose = false;
    _setStatus(WsConnectionStatus.connecting);

    try {
      final wsUrl = _buildWsUrl();
      debugPrint('[WsConnection] Connecting to $wsUrl');

      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));

      // Wait for the connection to be established.
      await _channel!.ready;

      _channelSubscription = _channel!.stream.listen(
        _onData,
        onError: _onError,
        onDone: _onDone,
      );

      _setStatus(WsConnectionStatus.connected);
      _reconnectAttempts = 0;
      debugPrint('[WsConnection] Connected to Bridge');
    } on Exception catch (e) {
      debugPrint('[WsConnection] Connection failed: $e');
      _setStatus(WsConnectionStatus.error);
      _scheduleReconnect();
    }
  }

  /// Sends a JSON message to Bridge.
  void send(Map<String, dynamic> message) {
    if (_channel == null || _status != WsConnectionStatus.connected) {
      debugPrint('[WsConnection] Cannot send: not connected');
      return;
    }

    final encoded = jsonEncode(message);
    _channel!.sink.add(encoded);
  }

  /// Gracefully disconnects from Bridge.
  Future<void> disconnect() async {
    _intentionalClose = true;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    await _channelSubscription?.cancel();
    _channelSubscription = null;
    await _channel?.sink.close();
    _channel = null;
    _setStatus(WsConnectionStatus.disconnected);
  }

  /// Disposes all resources. After calling this, the connection cannot be
  /// reused.
  Future<void> dispose() async {
    if (_disposed) return;
    _disposed = true;
    await disconnect();
    await _messageController.close();
    await _statusController.close();
  }

  // -------------------------------------------------------------------------
  // Internal
  // -------------------------------------------------------------------------

  String _buildWsUrl() {
    // bridgeUrl may be "ws://host:port", "wss://host:port", or "host:port".
    final base = bridgeUrl.startsWith('ws') ? bridgeUrl : 'ws://$bridgeUrl';

    final uri = Uri.parse(base);
    return uri
        .replace(
          path: '/connect',
          queryParameters: {'server_id': serverId, 'device_id': deviceId},
        )
        .toString();
  }

  void _onData(dynamic data) {
    if (data is! String) return;

    try {
      final json = jsonDecode(data) as Map<String, dynamic>;
      _messageController.add(json);
    } on FormatException catch (e) {
      debugPrint('[WsConnection] Invalid JSON received: $e');
    }
  }

  void _onError(Object error) {
    debugPrint('[WsConnection] Stream error: $error');
    _setStatus(WsConnectionStatus.error);
    if (!_intentionalClose) {
      _scheduleReconnect();
    }
  }

  void _onDone() {
    debugPrint('[WsConnection] Connection closed');
    if (_intentionalClose || _disposed) {
      _setStatus(WsConnectionStatus.disconnected);
      return;
    }

    _setStatus(WsConnectionStatus.error);
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_disposed || _intentionalClose) return;

    _reconnectTimer?.cancel();
    final delay = _reconnectDelay(_reconnectAttempts);
    debugPrint(
      '[WsConnection] Reconnecting in ${delay}s '
      '(attempt ${_reconnectAttempts + 1})',
    );

    _reconnectTimer = Timer(Duration(seconds: delay), () {
      _reconnectAttempts++;
      connect();
    });
  }

  int _reconnectDelay(int attempts) {
    final delay = min(2 << attempts, _maxReconnectDelay);
    return delay;
  }

  void _setStatus(WsConnectionStatus newStatus) {
    if (_status == newStatus) return;
    _status = newStatus;
    if (!_statusController.isClosed) {
      _statusController.add(newStatus);
    }
  }
}

/// Connection status for a WebSocket connection to Bridge.
enum WsConnectionStatus { disconnected, connecting, connected, error }

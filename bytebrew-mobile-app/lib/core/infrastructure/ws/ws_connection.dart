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
///
/// Detects dead connections via two mechanisms:
/// 1. Stale check: if no data arrives for [_staleThresholdSeconds], reconnect.
/// 2. Pong watchdog: if no data arrives within [_pongTimeoutSeconds] after a
///    ping is sent, the connection is considered dead.
///
/// On some Android devices (e.g. Xiaomi/MIUI), the TCP receive path stops
/// working after ~30-40s even with active keepalive traffic. The stale
/// detection handles this by triggering a reconnect, and the CLI replays
/// missed events via lastEventId on re-subscribe.
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
  Timer? _pingTimer;
  Timer? _pongWatchdog;
  int _reconnectAttempts = 0;
  bool _disposed = false;
  bool _intentionalClose = false;

  /// Timestamp of the last data received from the server.
  /// Used to detect stale connections (Android TCP receive path death).
  DateTime _lastDataAt = DateTime.now();

  static const _maxReconnectDelay = 30;

  /// Ping interval to keep the connection alive through carrier NAT.
  /// Bridge sends keepalive pong every 5s, so we get data at least every 5s.
  /// Our own ping at 10s acts as a secondary keepalive.
  static const _pingIntervalSeconds = 10;

  /// If no data (including pong) arrives within this many seconds after a ping
  /// is sent, the connection is considered dead.
  static const _pongTimeoutSeconds = 10;

  /// A connection is considered stale if no data has been received for longer
  /// than this threshold. With bridge keepalive every 5s, missing 2 consecutive
  /// pongs (10s) means the connection is dead.
  static const _staleThresholdSeconds = 10;

  /// Current connection status.
  WsConnectionStatus get status => _status;

  /// Whether a reconnect timer is active (exponential backoff in progress).
  /// Used by [WsConnectionManager] to avoid resetting backoff by recreating
  /// the connection.
  bool get hasActiveReconnect => _reconnectTimer?.isActive ?? false;

  /// Whether the connection appears stale (no data for [_staleThresholdSeconds]).
  bool get isStale =>
      _status == WsConnectionStatus.connected &&
      DateTime.now().difference(_lastDataAt).inSeconds > _staleThresholdSeconds;

  /// Stream of parsed JSON messages from Bridge.
  Stream<Map<String, dynamic>> get messages => _messageController.stream;

  /// Stream of connection status changes.
  Stream<WsConnectionStatus> get statusChanges => _statusController.stream;

  /// Connects to Bridge.
  Future<void> connect() async {
    if (_disposed) return;
    if (_status == WsConnectionStatus.connected) return;
    if (_status == WsConnectionStatus.connecting) return;

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
      // Don't reset _reconnectAttempts here — reset in _onData() after
      // receiving actual data. This prevents reconnect storms when the bridge
      // accepts the WS handshake but immediately closes (e.g. stale server_id).
      _lastDataAt = DateTime.now();
      _startPingLoop();
    } on Exception catch (e) {
      debugPrint('[WsConnection] Connection failed: $e');
      _setStatus(WsConnectionStatus.error);
      _scheduleReconnect();
    }
  }

  /// Verifies the connection is alive by sending a ping and waiting for any
  /// data to arrive. Returns `true` if the connection responded within
  /// [timeout], `false` if it appears dead.
  ///
  /// If data was received recently (within 10 seconds), returns `true`
  /// immediately without sending a probe.
  Future<bool> ensureAlive({
    Duration timeout = const Duration(seconds: 5),
  }) async {
    if (_status != WsConnectionStatus.connected || _channel == null) {
      return false;
    }

    // Recent data means the connection is alive.
    if (DateTime.now().difference(_lastDataAt).inSeconds < 10) return true;

    // Send a ping probe and wait for any response.
    try {
      _channel!.sink.add(jsonEncode({'type': 'ping'}));
    } on Object catch (_) {
      return false;
    }

    final before = _lastDataAt;
    final deadline = DateTime.now().add(timeout);
    while (DateTime.now().isBefore(deadline)) {
      await Future<void>.delayed(const Duration(milliseconds: 100));
      if (_lastDataAt.isAfter(before)) {
        return true;
      }
      if (_status != WsConnectionStatus.connected) return false;
    }

    _onPongTimeout();
    return false;
  }

  /// Sends a JSON message to Bridge.
  void send(Map<String, dynamic> message) {
    if (_channel == null || _status != WsConnectionStatus.connected) {
      debugPrint('[WsConnection] Cannot send: not connected (channel=${_channel != null} status=$_status)');
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
    _stopPingLoop();
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
    _stopPingLoop();
    await disconnect();
    await _messageController.close();
    await _statusController.close();
  }

  // -------------------------------------------------------------------------
  // Internal
  // -------------------------------------------------------------------------

  String _buildWsUrl() {
    // bridgeUrl may be "ws://host:port", "wss://host:port", or "host:port".
    // Port 443 implies TLS (wss://), others default to ws://.
    final String base;
    if (bridgeUrl.startsWith('ws')) {
      base = bridgeUrl;
    } else if (bridgeUrl.endsWith(':443') || bridgeUrl.startsWith('443')) {
      base = 'wss://$bridgeUrl';
    } else {
      base = 'ws://$bridgeUrl';
    }

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
    _lastDataAt = DateTime.now();

    // Data received means stable connection — reset backoff.
    _reconnectAttempts = 0;

    // Any data received means the connection is alive — reset the watchdog.
    _resetPongWatchdog();

    try {
      final json = jsonDecode(data) as Map<String, dynamic>;
      final type = json['type'] as String? ?? '?';

      // Filter out pong responses from bridge (keep-alive, not business data).
      if (type == 'pong') return;

      _messageController.add(json);
    } on FormatException catch (e) {
      debugPrint('[WsConnection] Invalid JSON received: $e');
    }
  }

  void _onError(Object error) {
    debugPrint('[WsConnection] Stream error: $error');
    _stopPingLoop();
    _setStatus(WsConnectionStatus.error);
    if (!_intentionalClose) {
      _scheduleReconnect();
    }
  }

  void _onDone() {
    debugPrint('[WsConnection] Connection closed');
    _stopPingLoop();
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
    _reconnectTimer = Timer(Duration(seconds: delay), () {
      _reconnectAttempts++;
      connect();
    });
  }

  /// Exponential backoff: 0s for first attempt, then 2, 4, 8, ... up to 30s.
  int _reconnectDelay(int attempts) {
    if (attempts == 0) return 0;
    return min(2 << (attempts - 1), _maxReconnectDelay);
  }

  /// Sends application-level ping to keep the connection alive through NAT.
  void _startPingLoop() {
    _stopPingLoop();
    _pingTimer = Timer.periodic(
      const Duration(seconds: _pingIntervalSeconds),
      (_) => _sendPing(),
    );
  }

  void _stopPingLoop() {
    _pingTimer?.cancel();
    _pingTimer = null;
    _pongWatchdog?.cancel();
    _pongWatchdog = null;
  }

  void _sendPing() {
    if (_channel == null || _status != WsConnectionStatus.connected) return;

    // If no data has been received for too long, the connection is dead.
    // Skip the ping and trigger immediate reconnect.
    if (isStale) {
      _onPongTimeout();
      return;
    }

    try {
      _channel!.sink.add(jsonEncode({'type': 'ping'}));
      _startPongWatchdog();
    } on Object catch (_) {}
  }

  /// Starts (or restarts) a watchdog timer. If no data arrives within
  /// [_pongTimeoutSeconds], the connection is declared dead.
  void _startPongWatchdog() {
    _pongWatchdog?.cancel();
    _pongWatchdog = Timer(
      const Duration(seconds: _pongTimeoutSeconds),
      _onPongTimeout,
    );
  }

  /// Resets the watchdog because data was received.
  void _resetPongWatchdog() {
    // Only reset if the watchdog is active (i.e., we're waiting for data).
    if (_pongWatchdog != null && _pongWatchdog!.isActive) {
      _pongWatchdog!.cancel();
      _pongWatchdog = null;
    }
  }

  /// Called when no data arrives within the pong timeout window.
  void _onPongTimeout() {
    _stopPingLoop();

    // Close the dead channel.
    _channelSubscription?.cancel();
    _channelSubscription = null;
    try {
      _channel?.sink.close();
    } on Object catch (_) {}
    _channel = null;

    _setStatus(WsConnectionStatus.error);
    _scheduleReconnect();
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

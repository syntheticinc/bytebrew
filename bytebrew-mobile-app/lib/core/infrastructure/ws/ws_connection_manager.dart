import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/foundation.dart';

import 'package:bytebrew_mobile/core/crypto/message_cipher.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

/// Connection status for a WebSocket connection via Bridge.
enum WsConnectionStatus { disconnected, connecting, connected, error }

// ---------------------------------------------------------------------------
// WsServerConnection
// ---------------------------------------------------------------------------

/// Represents an active WS connection to a paired CLI server via Bridge.
class WsServerConnection {
  WsServerConnection({
    required this.server,
    required this.connection,
    required this.client,
    this.cipher,
  });

  final Server server;
  final WsConnection connection;
  final WsBridgeClient client;

  /// E2E cipher for this connection (null if encryption not configured).
  final MessageCipher? cipher;

  /// The current connection status.
  WsConnectionStatus status = WsConnectionStatus.disconnected;

  /// The last error message, if any.
  String? lastError;

  /// Timer for scheduled reconnection attempts.
  Timer? reconnectTimer;

  /// Number of reconnect attempts since last successful connection.
  int reconnectAttempts = 0;
}

// ---------------------------------------------------------------------------
// WsConnectionManager
// ---------------------------------------------------------------------------

/// Manages WebSocket connections to paired CLI servers via Bridge.
///
/// Extends [ChangeNotifier] so UI widgets can react to connection state
/// changes via listeners.
class WsConnectionManager extends ChangeNotifier {
  WsConnectionManager({
    WsConnection Function({
      required String bridgeUrl,
      required String serverId,
      required String deviceId,
    })?
    connectionFactory,
  }) : _connectionFactory = connectionFactory;

  final WsConnection Function({
    required String bridgeUrl,
    required String serverId,
    required String deviceId,
  })?
  _connectionFactory;

  final Map<String, WsServerConnection> _connections = {};

  /// All current connections, keyed by server ID.
  Map<String, WsServerConnection> get connections =>
      Map.unmodifiable(_connections);

  /// Only connections with [WsConnectionStatus.connected].
  Iterable<WsServerConnection> get activeConnections => _connections.values
      .where((c) => c.status == WsConnectionStatus.connected);

  /// Returns the connection for [serverId], or `null` if not present.
  WsServerConnection? getConnection(String serverId) => _connections[serverId];

  // -----------------------------------------------------------------------
  // Connect / disconnect
  // -----------------------------------------------------------------------

  /// Connects to a single [server] via Bridge.
  ///
  /// Skips servers that have no [Server.deviceToken] or are already connected.
  /// Verifies the connection via ping before marking as connected.
  Future<void> connectToServer(Server server) async {
    if (server.deviceToken == null || server.deviceToken!.isEmpty) {
      return;
    }

    // Skip if already connected.
    final existing = _connections[server.id];
    if (existing != null && existing.status == WsConnectionStatus.connected) {
      return;
    }

    final wsConnection = _connectionFactory != null
        ? _connectionFactory(
            bridgeUrl: server.bridgeUrl,
            serverId: server.id,
            deviceId: server.deviceId ?? '',
          )
        : WsConnection(
            bridgeUrl: server.bridgeUrl,
            serverId: server.id,
            deviceId: server.deviceId ?? '',
          );

    final cipher = server.hasEncryption
        ? MessageCipher(server.sharedSecret!)
        : null;

    final client = WsBridgeClient(
      connection: wsConnection,
      cipher: cipher,
      deviceId: server.deviceId ?? '',
    );

    final serverConn = WsServerConnection(
      server: server,
      connection: wsConnection,
      client: client,
      cipher: cipher,
    );
    serverConn.status = WsConnectionStatus.connecting;
    _connections[server.id] = serverConn;
    notifyListeners();

    try {
      await wsConnection.connect();
      await client.ping();
      serverConn.status = WsConnectionStatus.connected;
      serverConn.reconnectAttempts = 0;
      serverConn.lastError = null;
    } on Exception catch (e) {
      serverConn.status = WsConnectionStatus.error;
      serverConn.lastError = e.toString();
    }

    notifyListeners();
  }

  /// Connects to all [servers].
  Future<void> connectToAll(List<Server> servers) async {
    for (final server in servers) {
      await connectToServer(server);
    }
  }

  /// Disconnects from the server identified by [serverId].
  Future<void> disconnectFromServer(String serverId) async {
    final serverConn = _connections.remove(serverId);
    if (serverConn == null) return;

    serverConn.reconnectTimer?.cancel();
    await serverConn.client.dispose();
    await serverConn.connection.dispose();
    notifyListeners();
  }

  /// Disconnects from all servers and clears all connections.
  Future<void> disconnectAll() async {
    for (final serverConn in _connections.values) {
      serverConn.reconnectTimer?.cancel();
      await serverConn.client.dispose();
      await serverConn.connection.dispose();
    }
    _connections.clear();
    notifyListeners();
  }

  @override
  void dispose() {
    for (final serverConn in _connections.values) {
      serverConn.reconnectTimer?.cancel();
    }
    _connections.clear();
    super.dispose();
  }

  // -----------------------------------------------------------------------
  // Health check & reconnect
  // -----------------------------------------------------------------------

  /// Marks a connection as lost and schedules a reconnection attempt.
  void markConnectionLost(String serverId, {String? reason}) {
    final serverConn = _connections[serverId];
    if (serverConn == null) return;

    if (serverConn.status != WsConnectionStatus.connected) return;

    serverConn.status = WsConnectionStatus.error;
    serverConn.lastError = reason;
    _scheduleReconnect(serverConn);
    notifyListeners();
  }

  void _scheduleReconnect(WsServerConnection serverConn) {
    serverConn.reconnectTimer?.cancel();

    final delay = Duration(
      seconds: _reconnectDelay(serverConn.reconnectAttempts),
    );

    serverConn.reconnectTimer = Timer(delay, () async {
      serverConn.reconnectAttempts++;
      await _attemptReconnect(serverConn);
    });
  }

  int _reconnectDelay(int attempts) {
    // Exponential backoff: 2, 4, 8, 16, 30 (capped at 30 seconds).
    final delay = 2 << attempts;
    return delay > 30 ? 30 : delay;
  }

  Future<void> _attemptReconnect(WsServerConnection serverConn) async {
    serverConn.status = WsConnectionStatus.connecting;
    notifyListeners();

    try {
      await serverConn.connection.connect();
      await serverConn.client.ping();
      serverConn.status = WsConnectionStatus.connected;
      serverConn.reconnectAttempts = 0;
      serverConn.lastError = null;
    } on Exception catch (e) {
      serverConn.status = WsConnectionStatus.error;
      serverConn.lastError = e.toString();
      _scheduleReconnect(serverConn);
    }

    notifyListeners();
  }

  // -----------------------------------------------------------------------
  // Encryption helpers
  // -----------------------------------------------------------------------

  /// Encrypts [data] for the given server.
  Future<Uint8List> encryptForServer(
    String serverId,
    Uint8List data,
    int counter,
  ) async {
    final serverConn = _connections[serverId];
    if (serverConn?.cipher == null) return data;

    return serverConn!.cipher!.encrypt(data, counter);
  }

  /// Decrypts [data] from the given server.
  Future<(Uint8List, int)> decryptFromServer(
    String serverId,
    Uint8List data,
  ) async {
    final serverConn = _connections[serverId];
    if (serverConn?.cipher == null) return (data, 0);

    return serverConn!.cipher!.decrypt(data);
  }
}

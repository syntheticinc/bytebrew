import 'dart:async';

import 'package:flutter/foundation.dart';

import 'package:bytebrew_mobile/core/crypto/message_cipher.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';

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

  /// Subscription to WsConnection status changes.
  StreamSubscription<WsConnectionStatus>? statusSubscription;
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

    // Clean up any existing connection before creating a new one.
    // Always recreate to pick up fresh crypto keys after re-pairing.
    final existing = _connections[server.id];
    if (existing != null) {
      await existing.statusSubscription?.cancel();
      await existing.client.dispose();
      await existing.connection.dispose();
      _connections.remove(server.id);
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

    // Mirror WsConnection status changes 1:1 into serverConn.
    // WsConnection owns reconnect logic; manager is a thin status mirror.
    serverConn.statusSubscription = wsConnection.statusChanges.listen((s) {
      final conn = _connections[server.id];
      if (conn == null || conn != serverConn) return;

      conn.status = s;
      if (s == WsConnectionStatus.error) {
        conn.lastError = 'Connection lost';
      } else if (s == WsConnectionStatus.connected) {
        conn.lastError = null;
      }
      notifyListeners();
    });

    notifyListeners();

    try {
      await wsConnection.connect();
      await client.ping();
      serverConn.status = WsConnectionStatus.connected;
      serverConn.lastError = null;
    } on Exception catch (e) {
      serverConn.lastError = e.toString();
      // If WS socket opened but CLI unreachable (ping fail) —
      // disconnect so auto-connect retry via 30s timer.
      if (wsConnection.status == WsConnectionStatus.connected) {
        await wsConnection.disconnect();
      }
      serverConn.status = WsConnectionStatus.error;
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

    await serverConn.statusSubscription?.cancel();
    await serverConn.client.dispose();
    await serverConn.connection.dispose();
    notifyListeners();
  }

  /// Disconnects from all servers and clears all connections.
  Future<void> disconnectAll() async {
    for (final serverConn in _connections.values) {
      await serverConn.statusSubscription?.cancel();
      await serverConn.client.dispose();
      await serverConn.connection.dispose();
    }
    _connections.clear();
    notifyListeners();
  }

  @override
  void dispose() {
    _connections.clear();
    super.dispose();
  }
}

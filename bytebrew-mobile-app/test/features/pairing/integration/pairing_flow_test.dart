import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' hide Server;

import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/grpc_channel_factory.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/mobile_service_client.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:bytebrew_mobile/features/settings/infrastructure/local_settings_repository.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Configurable fake [MobileServiceClient] for integration tests.
///
/// Supports configurable ping, pair, and listSessions responses.
/// Tracks all method calls for assertions.
class _FakeMobileServiceClient implements MobileServiceClient {
  bool pingCalled = false;
  bool closeCalled = false;
  int pingCallCount = 0;

  /// If non-null, [ping] will throw this error.
  Object? pingError;

  /// Configurable ping result.
  PingResult? pingResult;

  /// Configurable pair result.
  PairResult pairResultToReturn = PairResult(
    deviceId: 'device-123',
    deviceToken: 'token-abc',
    serverName: 'Dev Workstation',
    serverId: 'srv-456',
    serverPublicKey: null,
  );

  /// Captured pair calls.
  final pairCalls =
      <({String token, String deviceName, Uint8List? mobilePublicKey})>[];

  /// If non-null, [pair] will throw this error.
  Object? pairError;

  /// Sessions to return from [listSessions].
  List<MobileSession> sessionsToReturn = [];

  @override
  Future<PingResult> ping() async {
    pingCalled = true;
    pingCallCount++;
    if (pingError != null) throw pingError!;
    return pingResult ??
        PingResult(
          timestamp: DateTime.now(),
          serverName: 'Test Server',
          serverId: 'test-server-id',
        );
  }

  @override
  Future<PairResult> pair({
    required String token,
    required String deviceName,
    Uint8List? mobilePublicKey,
  }) async {
    pairCalls.add((
      token: token,
      deviceName: deviceName,
      mobilePublicKey: mobilePublicKey,
    ));
    if (pairError != null) throw pairError!;
    return pairResultToReturn;
  }

  @override
  Future<void> close() async {
    closeCalled = true;
  }

  @override
  Future<ListSessionsResult> listSessions({
    required String deviceToken,
  }) async {
    return ListSessionsResult(
      sessions: sessionsToReturn,
      serverName: 'Test Server',
      serverId: 'test-server-id',
    );
  }

  @override
  Stream<SessionEvent> subscribeSession({
    required String deviceToken,
    required String sessionId,
    String? lastEventId,
  }) {
    return const Stream.empty();
  }

  @override
  Future<SendCommandResult> sendNewTask({
    required String deviceToken,
    required String sessionId,
    required String task,
  }) async {
    return const SendCommandResult(success: true);
  }

  @override
  Future<SendCommandResult> sendAskUserReply({
    required String deviceToken,
    required String sessionId,
    required String question,
    required String answer,
  }) async {
    return const SendCommandResult(success: true);
  }

  @override
  Future<SendCommandResult> cancelSession({
    required String deviceToken,
    required String sessionId,
  }) async {
    return const SendCommandResult(success: true);
  }
}

/// Fake [GrpcChannelFactory] that returns dummy channels.
class _FakeGrpcChannelFactory implements GrpcChannelFactory {
  final createdChannels = <Server>[];

  @override
  ClientChannel createChannel(Server server) {
    createdChannels.add(server);
    return ClientChannel(
      'localhost',
      port: 1,
      options: const ChannelOptions(
        credentials: ChannelCredentials.insecure(),
      ),
    );
  }

  @override
  ClientChannel createBridgeChannel(String bridgeUrl) {
    return ClientChannel(
      'localhost',
      port: 1,
      options: const ChannelOptions(
        credentials: ChannelCredentials.insecure(),
      ),
    );
  }
}

/// In-memory fake of [LocalSettingsRepository] that tracks additions
/// and can return stored servers.
class _FakeLocalSettingsRepository implements LocalSettingsRepository {
  final addedServers = <Server>[];
  final removedServerIds = <String>[];
  List<Server> _servers = [];

  @override
  Future<void> addServer(Server server) async {
    addedServers.add(server);
    final idx = _servers.indexWhere((s) => s.id == server.id);
    if (idx != -1) {
      _servers[idx] = server;
    } else {
      _servers.add(server);
    }
  }

  @override
  List<Server> getServers() => List.unmodifiable(_servers);

  @override
  Future<List<Server>> getServersWithKeys() async =>
      List.unmodifiable(_servers);

  @override
  Future<void> removeServer(String id) async {
    removedServerIds.add(id);
    _servers = _servers.where((s) => s.id != id).toList();
  }
}

/// Fake [PairingRepository] that delegates to a [_FakeMobileServiceClient]
/// and persists to a [_FakeLocalSettingsRepository].
///
/// Simulates the real [GrpcPairingRepository] without crypto or gRPC.
class _FakePairingRepository implements PairingRepository {
  _FakePairingRepository({
    required this.client,
    required this.settingsRepo,
  });

  final _FakeMobileServiceClient client;
  final _FakeLocalSettingsRepository settingsRepo;

  @override
  Future<Server> pair({
    required String serverAddress,
    required String pairingCode,
  }) async {
    final parts = serverAddress.split(':');
    final host = parts[0];
    final port = parts.length > 1 ? int.tryParse(parts[1]) ?? 60401 : 60401;

    final result = await client.pair(
      token: pairingCode,
      deviceName: 'Flutter Mobile',
    );

    final server = Server(
      id: result.serverId.isNotEmpty
          ? result.serverId
          : 'srv-${DateTime.now().millisecondsSinceEpoch}',
      name: result.serverName.isNotEmpty ? result.serverName : host,
      lanAddress: host,
      connectionMode: ConnectionMode.lan,
      isOnline: true,
      latencyMs: 0,
      pairedAt: DateTime.now(),
      deviceId: result.deviceId,
      deviceToken: result.deviceToken,
      grpcPort: port,
    );

    await settingsRepo.addServer(server);
    return server;
  }
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/// Standard paired server for reuse across tests.
Server _makePairedServer({
  String id = 'srv-456',
  String name = 'Dev Workstation',
  String lanAddress = '192.168.1.14',
  String deviceToken = 'token-abc',
  int grpcPort = 60466,
}) {
  return Server(
    id: id,
    name: name,
    lanAddress: lanAddress,
    connectionMode: ConnectionMode.lan,
    isOnline: true,
    latencyMs: 0,
    pairedAt: DateTime.now(),
    deviceId: 'device-123',
    deviceToken: deviceToken,
    grpcPort: grpcPort,
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('Pairing integration flow', () {
    late _FakeMobileServiceClient fakeClient;
    late _FakeLocalSettingsRepository fakeSettingsRepo;
    late _FakePairingRepository pairingRepo;
    late ConnectionManager connectionManager;

    setUp(() {
      fakeClient = _FakeMobileServiceClient();
      fakeSettingsRepo = _FakeLocalSettingsRepository();
      pairingRepo = _FakePairingRepository(
        client: fakeClient,
        settingsRepo: fakeSettingsRepo,
      );
      connectionManager = ConnectionManager(
        clientFactory: (_) => fakeClient,
      );
    });

    tearDown(() async {
      await connectionManager.disconnectAll();
    });

    // -----------------------------------------------------------------------
    // Test 1: pair() saves server to settings
    // -----------------------------------------------------------------------
    test('pair() saves server to local settings with correct fields', () async {
      fakeClient.pairResultToReturn = PairResult(
        deviceId: 'dev-1',
        deviceToken: 'tok-1',
        serverName: 'My Workstation',
        serverId: 'srv-1',
        serverPublicKey: null,
      );

      final server = await pairingRepo.pair(
        serverAddress: '192.168.1.14:60466',
        pairingCode: 'token123',
      );

      // Server returned from pair() has correct fields.
      expect(server.id, 'srv-1');
      expect(server.name, 'My Workstation');
      expect(server.lanAddress, '192.168.1.14');
      expect(server.deviceToken, 'tok-1');
      expect(server.deviceId, 'dev-1');
      expect(server.grpcPort, 60466);

      // Server was persisted to settings.
      expect(fakeSettingsRepo.addedServers, hasLength(1));
      final saved = fakeSettingsRepo.addedServers.first;
      expect(saved.id, server.id);
      expect(saved.name, 'My Workstation');
      expect(saved.deviceToken, 'tok-1');
      expect(saved.lanAddress, '192.168.1.14');
      expect(saved.grpcPort, 60466);

      // Settings repo returns the server via getServers().
      expect(fakeSettingsRepo.getServers(), hasLength(1));
      expect(fakeSettingsRepo.getServers().first.id, 'srv-1');
    });

    // -----------------------------------------------------------------------
    // Test 2: connectToServer establishes LAN connection
    // -----------------------------------------------------------------------
    test('connectToServer establishes LAN connection via ping', () async {
      fakeClient.pingResult = PingResult(
        timestamp: DateTime.now(),
        serverName: 'Dev Workstation',
        serverId: 'srv-456',
      );

      final server = _makePairedServer();
      await connectionManager.connectToServer(server);

      final connection = connectionManager.getConnection('srv-456');
      expect(connection, isNotNull);
      expect(connection!.status, GrpcConnectionStatus.connected);
      expect(connection.currentRoute, ConnectionRoute.lan);
      expect(fakeClient.pingCalled, isTrue);
    });

    // -----------------------------------------------------------------------
    // Test 3: connectToServer skips already connected server
    // -----------------------------------------------------------------------
    test('connectToServer skips server that is already connected', () async {
      final server = _makePairedServer();

      // First connection.
      await connectionManager.connectToServer(server);

      final connection1 = connectionManager.getConnection('srv-456');
      expect(connection1, isNotNull);
      expect(connection1!.status, GrpcConnectionStatus.connected);
      expect(fakeClient.pingCallCount, 1);

      // Second connection attempt -- should be skipped.
      await connectionManager.connectToServer(server);

      // Ping count should NOT increase (connection was skipped).
      expect(fakeClient.pingCallCount, 1);

      // Connection is still active.
      final connection2 = connectionManager.getConnection('srv-456');
      expect(connection2, isNotNull);
      expect(connection2!.status, GrpcConnectionStatus.connected);
    });

    // -----------------------------------------------------------------------
    // Test 4: Auto-connect does not overwrite existing connection
    // -----------------------------------------------------------------------
    test('auto-connect does not tear down existing connection', () async {
      final server = _makePairedServer();

      // Establish connection via pairing flow.
      await connectionManager.connectToServer(server);
      expect(
        connectionManager.getConnection('srv-456')!.status,
        GrpcConnectionStatus.connected,
      );
      expect(fakeClient.pingCallCount, 1);

      // Simulate auto-connect: reads servers from settings, calls connectToAll.
      await fakeSettingsRepo.addServer(server);
      final servers = await fakeSettingsRepo.getServersWithKeys();
      await connectionManager.connectToAll(servers);

      // Connection should still be active -- NOT torn down and recreated.
      expect(fakeClient.pingCallCount, 1);
      expect(
        connectionManager.getConnection('srv-456')!.status,
        GrpcConnectionStatus.connected,
      );
    });

    // -----------------------------------------------------------------------
    // Test 5: Full pairing -> sessions flow (end-to-end)
    // -----------------------------------------------------------------------
    test('full flow: pair -> connect -> list sessions', () async {
      // Configure pair response.
      fakeClient.pairResultToReturn = PairResult(
        deviceId: 'dev-e2e',
        deviceToken: 'tok-e2e',
        serverName: 'E2E Workstation',
        serverId: 'srv-e2e',
        serverPublicKey: null,
      );

      // Configure sessions response.
      fakeClient.sessionsToReturn = [
        MobileSession(
          sessionId: 'session-1',
          projectKey: 'my-project',
          projectRoot: '/home/user/my-project',
          status: MobileSessionState.active,
          currentTask: 'Refactor auth module',
          startedAt: DateTime.now().subtract(const Duration(hours: 1)),
          lastActivityAt: DateTime.now(),
          hasAskUser: false,
          platform: 'linux',
        ),
        MobileSession(
          sessionId: 'session-2',
          projectKey: 'other-project',
          projectRoot: '/home/user/other-project',
          status: MobileSessionState.idle,
          currentTask: '',
          startedAt: DateTime.now().subtract(const Duration(hours: 2)),
          lastActivityAt: DateTime.now().subtract(const Duration(minutes: 30)),
          hasAskUser: false,
          platform: 'linux',
        ),
      ];

      // Step 1: Pair.
      final server = await pairingRepo.pair(
        serverAddress: '192.168.1.14:60466',
        pairingCode: 'token123',
      );
      expect(server.deviceToken, 'tok-e2e');

      // Step 2: Connect.
      await connectionManager.connectToServer(server);
      expect(
        connectionManager.getConnection('srv-e2e')!.status,
        GrpcConnectionStatus.connected,
      );

      // Step 3: List sessions from the connected server.
      final sessions = await connectionManager.listAllSessions();
      expect(sessions, hasLength(2));
      expect(sessions[0].sessionId, 'session-1');
      expect(sessions[0].currentTask, 'Refactor auth module');
      expect(sessions[1].sessionId, 'session-2');
    });

    // -----------------------------------------------------------------------
    // Test 6: Settings shows paired server after pair
    // -----------------------------------------------------------------------
    test('settings repo returns paired server with correct data', () async {
      fakeClient.pairResultToReturn = PairResult(
        deviceId: 'dev-set',
        deviceToken: 'tok-set',
        serverName: 'Settings Server',
        serverId: 'srv-set',
        serverPublicKey: null,
      );

      await pairingRepo.pair(
        serverAddress: '10.0.0.5:8080',
        pairingCode: 'code99',
      );

      // Settings returns the paired server.
      final servers = fakeSettingsRepo.getServers();
      expect(servers, hasLength(1));

      final saved = servers.first;
      expect(saved.id, 'srv-set');
      expect(saved.name, 'Settings Server');
      expect(saved.lanAddress, '10.0.0.5');
      expect(saved.grpcPort, 8080);
      expect(saved.deviceToken, 'tok-set');
      expect(saved.deviceId, 'dev-set');
      expect(saved.connectionMode, ConnectionMode.lan);
    });

    // -----------------------------------------------------------------------
    // Test 7: Double scan prevention
    // -----------------------------------------------------------------------
    test('double scan prevention: _isLoading blocks second _onQrScanned',
        () async {
      // This test verifies the double-scan guard in AddServerScreen.
      //
      // The widget sets `_isLoading = true` synchronously before any
      // async work. A second call to `_onQrScanned` while `_isLoading`
      // is true returns immediately without calling pair().
      //
      // We simulate this at the logical level: a guard flag prevents
      // concurrent pairing.

      var isLoading = false;
      var pairCallCount = 0;

      // Simulates _onQrScanned with the same guard logic as the widget.
      Future<void> onQrScanned() async {
        if (isLoading) return; // Double-scan guard.
        isLoading = true;

        try {
          pairCallCount++;
          await pairingRepo.pair(
            serverAddress: '192.168.1.14:60466',
            pairingCode: 'token123',
          );
        } finally {
          isLoading = false;
        }
      }

      // First scan -- starts pairing, sets isLoading.
      final firstScan = onQrScanned();

      // Second scan -- immediately blocked by isLoading guard.
      // Note: in the real widget, _isLoading is set synchronously before
      // any await, so the second callback on the same frame is blocked.
      await onQrScanned();

      await firstScan;

      // Only one pair call should have been made.
      expect(pairCallCount, 1);
      expect(fakeClient.pairCalls, hasLength(1));
    });

    // -----------------------------------------------------------------------
    // Additional: connection status transitions are emitted
    // -----------------------------------------------------------------------
    test('connection notifies listeners during connect', () async {
      final server = _makePairedServer();
      var notifyCount = 0;
      connectionManager.addListener(() => notifyCount++);

      await connectionManager.connectToServer(server);

      // At minimum: connecting + connected = 2 notifications.
      expect(notifyCount, greaterThanOrEqualTo(2));
    });

    // -----------------------------------------------------------------------
    // Additional: pair failure does not persist server
    // -----------------------------------------------------------------------
    test('pair failure does not persist server to settings', () async {
      fakeClient.pairError = GrpcError.unauthenticated('bad token');

      try {
        await pairingRepo.pair(
          serverAddress: '192.168.1.14:60466',
          pairingCode: 'bad-code',
        );
        fail('Expected GrpcError');
      } on GrpcError catch (e) {
        expect(e.code, StatusCode.unauthenticated);
      }

      // No server should be saved.
      expect(fakeSettingsRepo.addedServers, isEmpty);
      expect(fakeSettingsRepo.getServers(), isEmpty);
    });

    // -----------------------------------------------------------------------
    // Additional: connectToServer without device token is a no-op
    // -----------------------------------------------------------------------
    test('connectToServer without device token does not connect', () async {
      final server = Server(
        id: 'srv-notoken',
        name: 'No Token',
        lanAddress: '192.168.1.100',
        connectionMode: ConnectionMode.lan,
        isOnline: false,
        latencyMs: 0,
        pairedAt: DateTime.now(),
      );

      await connectionManager.connectToServer(server);

      expect(connectionManager.getConnection('srv-notoken'), isNull);
      expect(fakeClient.pingCalled, isFalse);
    });

    // -----------------------------------------------------------------------
    // Additional: multiple servers can connect independently
    // -----------------------------------------------------------------------
    test('multiple servers can be connected independently', () async {
      final server1 = _makePairedServer(
        id: 'srv-1',
        name: 'Server A',
        deviceToken: 'tok-a',
      );
      final server2 = _makePairedServer(
        id: 'srv-2',
        name: 'Server B',
        deviceToken: 'tok-b',
      );

      await connectionManager.connectToServer(server1);
      await connectionManager.connectToServer(server2);

      expect(connectionManager.activeConnections, hasLength(2));
      expect(
        connectionManager.getConnection('srv-1')!.status,
        GrpcConnectionStatus.connected,
      );
      expect(
        connectionManager.getConnection('srv-2')!.status,
        GrpcConnectionStatus.connected,
      );
    });

    // -----------------------------------------------------------------------
    // Additional: connect -> disconnect -> reconnect
    // -----------------------------------------------------------------------
    test('server can be disconnected and reconnected', () async {
      final server = _makePairedServer();

      // Connect.
      await connectionManager.connectToServer(server);
      expect(
        connectionManager.getConnection('srv-456')!.status,
        GrpcConnectionStatus.connected,
      );

      // Disconnect.
      await connectionManager.disconnectFromServer('srv-456');
      expect(connectionManager.getConnection('srv-456'), isNull);

      // Reset ping counter for reconnect.
      fakeClient.pingCallCount = 0;

      // Reconnect.
      await connectionManager.connectToServer(server);
      expect(
        connectionManager.getConnection('srv-456')!.status,
        GrpcConnectionStatus.connected,
      );
      expect(fakeClient.pingCallCount, 1);
    });
  });
}

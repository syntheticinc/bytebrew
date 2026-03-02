import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' hide Server;

import 'package:bytebrew_mobile/core/crypto/key_exchange.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/grpc_channel_factory.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/mobile_service_client.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:bytebrew_mobile/features/pairing/infrastructure/grpc_pairing_repository.dart';
import 'package:bytebrew_mobile/features/settings/infrastructure/local_settings_repository.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Fake [MobileServiceClient] that captures pair calls and returns
/// configurable results.
class FakeMobileServiceClient implements MobileServiceClient {
  bool closeCalled = false;

  /// The pair result to return.
  PairResult pairResultToReturn = PairResult(
    deviceId: 'device-123',
    deviceToken: 'token-abc',
    serverName: 'Dev Workstation',
    serverId: 'srv-456',
    serverPublicKey: null,
  );

  /// Captured pair calls for assertions.
  final pairCalls = <({String token, String deviceName, Uint8List? mobilePublicKey})>[];

  /// If non-null, [pair] will throw this error.
  Object? pairError;

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
  Future<PingResult> ping() async {
    return PingResult(
      timestamp: DateTime.now(),
      serverName: 'Test',
      serverId: 'test-id',
    );
  }

  @override
  Future<ListSessionsResult> listSessions({
    required String deviceToken,
  }) async {
    return const ListSessionsResult(
      sessions: [],
      serverName: 'Test',
      serverId: 'test-id',
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

/// Fake [GrpcChannelFactory] that returns a dummy channel and records
/// the server it was called with.
class FakeGrpcChannelFactory implements GrpcChannelFactory {
  final createdChannels = <Server>[];

  @override
  ClientChannel createChannel(Server server) {
    createdChannels.add(server);
    // Return a real channel pointed at localhost that will never be used
    // because we intercept at the client level.
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

/// Fake [KeyExchange] that returns deterministic key material.
class FakeKeyExchange implements KeyExchange {
  static final fakePublicKeyBytes = Uint8List.fromList(
    List.generate(32, (i) => i + 1),
  );

  static final fakeSharedSecretBytes = Uint8List.fromList(
    List.generate(32, (i) => i + 100),
  );

  late SimpleKeyPair _keyPair;
  bool generateKeypairCalled = false;
  bool computeSharedSecretCalled = false;

  @override
  Future<SimpleKeyPair> generateKeypair() async {
    generateKeypairCalled = true;
    // Generate a real keypair for structural correctness.
    _keyPair = await X25519().newKeyPair();
    return _keyPair;
  }

  @override
  Future<Uint8List> extractPublicKeyBytes(SimpleKeyPair keyPair) async {
    return fakePublicKeyBytes;
  }

  @override
  Future<SecretKey> computeSharedSecret(
    SimpleKeyPair ours,
    SimplePublicKey theirs,
  ) async {
    computeSharedSecretCalled = true;
    return SecretKey(fakeSharedSecretBytes);
  }

  @override
  Future<SecretKey> computeSharedSecretFromBytes(
    SimpleKeyPair ourKeyPair,
    Uint8List theirPublicKeyBytes,
  ) async {
    computeSharedSecretCalled = true;
    return SecretKey(fakeSharedSecretBytes);
  }
}

/// Fake [LocalSettingsRepository] that records servers added to it.
class FakeLocalSettingsRepository implements LocalSettingsRepository {
  final addedServers = <Server>[];
  final removedServerIds = <String>[];

  @override
  Future<void> addServer(Server server) async {
    addedServers.add(server);
  }

  @override
  List<Server> getServers() => [];

  @override
  Future<List<Server>> getServersWithKeys() async => [];

  @override
  Future<void> removeServer(String id) async {
    removedServerIds.add(id);
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Creates a [GrpcPairingRepository] wired to the given fakes.
///
/// The [clientFactory] ignores the channel and returns [fakeClient],
/// so no real gRPC connection is ever made.
GrpcPairingRepository _createRepo({
  required FakeMobileServiceClient fakeClient,
  required FakeLocalSettingsRepository fakeSettingsRepo,
  required FakeKeyExchange fakeKeyExchange,
  FakeGrpcChannelFactory? fakeChannelFactory,
}) {
  return GrpcPairingRepository(
    settingsRepo: fakeSettingsRepo,
    channelFactory: fakeChannelFactory ?? FakeGrpcChannelFactory(),
    keyExchange: fakeKeyExchange,
    clientFactory: (_) => fakeClient,
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('GrpcPairingRepository', () {
    late FakeMobileServiceClient fakeClient;
    late FakeLocalSettingsRepository fakeSettingsRepo;
    late FakeKeyExchange fakeKeyExchange;
    late FakeGrpcChannelFactory fakeChannelFactory;
    late GrpcPairingRepository repo;

    setUp(() {
      fakeClient = FakeMobileServiceClient();
      fakeSettingsRepo = FakeLocalSettingsRepository();
      fakeKeyExchange = FakeKeyExchange();
      fakeChannelFactory = FakeGrpcChannelFactory();

      repo = _createRepo(
        fakeClient: fakeClient,
        fakeSettingsRepo: fakeSettingsRepo,
        fakeKeyExchange: fakeKeyExchange,
        fakeChannelFactory: fakeChannelFactory,
      );
    });

    group('pair() result mapping', () {
      test('returns server with correct fields from gRPC response', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: 'My Workstation',
          serverId: 'srv-789',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          serverAddress: '192.168.1.50',
          pairingCode: '123456',
        );

        expect(server.id, 'srv-789');
        expect(server.name, 'My Workstation');
        expect(server.lanAddress, '192.168.1.50');
        expect(server.deviceId, 'dev-123');
        expect(server.deviceToken, 'tok-456');
        expect(server.grpcPort, 60401);
        expect(server.isOnline, isTrue);
        expect(server.connectionMode, ConnectionMode.lan);
      });

      test('falls back to host as name when serverName is empty', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: '',
          serverId: 'srv-789',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          serverAddress: '10.0.0.5',
          pairingCode: '000000',
        );

        expect(server.name, '10.0.0.5');
      });

      test('generates fallback server ID when serverId is empty', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: 'Server',
          serverId: '',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          serverAddress: '10.0.0.1',
          pairingCode: '000000',
        );

        expect(server.id, startsWith('srv-'));
        expect(server.id, isNot(''));
      });
    });

    group('pair() address parsing', () {
      test('uses default port when no port specified', () async {
        final server = await repo.pair(
          serverAddress: '192.168.1.100',
          pairingCode: '123456',
        );

        expect(server.lanAddress, '192.168.1.100');
        expect(server.grpcPort, 60401);
      });

      test('parses host:port correctly', () async {
        final server = await repo.pair(
          serverAddress: '192.168.1.100:9090',
          pairingCode: '123456',
        );

        expect(server.lanAddress, '192.168.1.100');
        expect(server.grpcPort, 9090);
      });

      test('uses default port for invalid port string', () async {
        final server = await repo.pair(
          serverAddress: '192.168.1.100:notaport',
          pairingCode: '123456',
        );

        expect(server.lanAddress, '192.168.1.100');
        expect(server.grpcPort, 60401);
      });
    });

    group('pair() encryption / key exchange', () {
      test('computes shared secret when server returns public key', () async {
        final serverPubKey = Uint8List.fromList(
          List.generate(32, (i) => i + 50),
        );

        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: 'Secure Server',
          serverId: 'srv-1',
          serverPublicKey: serverPubKey,
        );

        final server = await repo.pair(
          serverAddress: '10.0.0.1',
          pairingCode: '999999',
        );

        // KeyExchange was invoked.
        expect(fakeKeyExchange.generateKeypairCalled, isTrue);
        expect(fakeKeyExchange.computeSharedSecretCalled, isTrue);

        // Server has encryption keys populated.
        expect(server.hasEncryption, isTrue);
        expect(server.sharedSecret, isNotNull);
        expect(server.sharedSecret!.length, 32);
        expect(server.sharedSecret, FakeKeyExchange.fakeSharedSecretBytes);
        expect(server.publicKey, FakeKeyExchange.fakePublicKeyBytes);
        expect(server.serverPublicKey, serverPubKey);
      });

      test('skips encryption when server returns no public key', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: 'Plain Server',
          serverId: 'srv-2',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          serverAddress: '10.0.0.1',
          pairingCode: '000000',
        );

        expect(fakeKeyExchange.computeSharedSecretCalled, isFalse);
        expect(server.hasEncryption, isFalse);
        expect(server.sharedSecret, isNull);
        expect(server.serverPublicKey, isNull);
      });

      test('skips encryption when server public key is too short', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: 'Bad Key Server',
          serverId: 'srv-3',
          serverPublicKey: Uint8List.fromList([1, 2, 3]),
        );

        final server = await repo.pair(
          serverAddress: '10.0.0.1',
          pairingCode: '000000',
        );

        expect(fakeKeyExchange.computeSharedSecretCalled, isFalse);
        expect(server.hasEncryption, isFalse);
      });
    });

    group('pair() persists to settings', () {
      test('saves paired server to local settings', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-1',
          deviceToken: 'tok-1',
          serverName: 'Saved Server',
          serverId: 'srv-saved',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          serverAddress: '192.168.1.10',
          pairingCode: '111111',
        );

        expect(fakeSettingsRepo.addedServers, hasLength(1));
        final saved = fakeSettingsRepo.addedServers.first;
        expect(saved.id, server.id);
        expect(saved.name, 'Saved Server');
        expect(saved.deviceToken, 'tok-1');
      });
    });

    group('pair() passes correct args to gRPC client', () {
      test('sends pairing code and device name', () async {
        await repo.pair(
          serverAddress: '192.168.1.1',
          pairingCode: '654321',
        );

        expect(fakeClient.pairCalls, hasLength(1));
        expect(fakeClient.pairCalls.first.token, '654321');
        expect(fakeClient.pairCalls.first.deviceName, 'Flutter Mobile');
      });

      test('sends mobile public key for key exchange', () async {
        await repo.pair(
          serverAddress: '192.168.1.1',
          pairingCode: '123456',
        );

        expect(fakeClient.pairCalls, hasLength(1));
        final sentKey = fakeClient.pairCalls.first.mobilePublicKey;
        expect(sentKey, isNotNull);
        expect(sentKey, FakeKeyExchange.fakePublicKeyBytes);
      });
    });

    group('pair() client lifecycle', () {
      test('closes gRPC client after successful pair', () async {
        await repo.pair(
          serverAddress: '192.168.1.1',
          pairingCode: '123456',
        );

        expect(fakeClient.closeCalled, isTrue);
      });

      test('closes gRPC client even when pair throws', () async {
        fakeClient.pairError = GrpcError.unauthenticated('bad token');

        try {
          await repo.pair(
            serverAddress: '192.168.1.1',
            pairingCode: 'bad',
          );
        } on GrpcError catch (_) {
          // Expected.
        }

        expect(fakeClient.closeCalled, isTrue);
      });
    });

    group('pair() error handling', () {
      test('propagates GrpcError from client', () async {
        fakeClient.pairError = GrpcError.unauthenticated('invalid token');

        await expectLater(
          repo.pair(serverAddress: '192.168.1.1', pairingCode: 'bad'),
          throwsA(isA<GrpcError>()),
        );
      });

      test('does not persist server when pair fails', () async {
        fakeClient.pairError = GrpcError.unavailable('server down');

        try {
          await repo.pair(
            serverAddress: '192.168.1.1',
            pairingCode: '000000',
          );
        } on GrpcError catch (_) {
          // Expected.
        }

        expect(fakeSettingsRepo.addedServers, isEmpty);
      });
    });

    group('PairingRepository interface', () {
      test('GrpcPairingRepository implements PairingRepository', () {
        expect(repo, isA<PairingRepository>());
      });
    });

    group('pair() creates channel with correct server', () {
      test('passes parsed host and port to channel factory', () async {
        await repo.pair(
          serverAddress: '10.0.0.5:8080',
          pairingCode: '123456',
        );

        expect(fakeChannelFactory.createdChannels, hasLength(1));
        final tempServer = fakeChannelFactory.createdChannels.first;
        expect(tempServer.lanAddress, '10.0.0.5');
        expect(tempServer.grpcPort, 8080);
      });
    });
  });
}

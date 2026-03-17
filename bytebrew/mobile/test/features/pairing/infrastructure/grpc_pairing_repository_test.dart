import 'dart:async';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/crypto/key_exchange.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/secure_key_storage.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:bytebrew_mobile/features/pairing/infrastructure/ws_pairing_repository.dart';
import 'package:bytebrew_mobile/features/settings/domain/settings_repository.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Fake [WsConnection] that avoids real WebSocket calls.
class _FakeWsConnection implements WsConnection {
  bool connectCalled = false;
  bool disposeCalled = false;

  /// If non-null, [connect] will throw this error.
  Object? connectError;

  @override
  Future<void> connect() async {
    connectCalled = true;
    if (connectError != null) throw connectError!;
  }

  @override
  Future<void> disconnect() async {}

  @override
  Future<void> dispose() async {
    disposeCalled = true;
  }

  @override
  Stream<Map<String, dynamic>> get messages => const Stream.empty();

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Fake [WsBridgeClient] that captures pair calls and returns
/// configurable results.
class _FakeWsBridgeClient implements WsBridgeClient {
  bool disposeCalled = false;

  /// The pair result to return.
  PairResult pairResultToReturn = PairResult(
    deviceId: 'device-123',
    deviceToken: 'token-abc',
    serverName: 'Dev Workstation',
    serverId: 'srv-456',
    serverPublicKey: null,
  );

  /// Captured pair calls for assertions.
  final pairCalls =
      <({String token, String deviceName, Uint8List? mobilePublicKey})>[];

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
  Future<void> dispose() async {
    disposeCalled = true;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Fake [KeyExchange] that returns deterministic key material.
class _FakeKeyExchange implements KeyExchange {
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

/// In-memory fake [SecureKeyStorage] that avoids platform channels.
class _FakeSecureKeyStorage implements SecureKeyStorage {
  final savedKeys = <String, Map<String, Uint8List>>{};
  final savedTokens = <String, String>{};

  @override
  Future<void> saveServerKeys({
    required String serverId,
    required Uint8List sharedSecret,
    Uint8List? publicKey,
    Uint8List? serverPublicKey,
  }) async {
    final keys = <String, Uint8List>{'sharedSecret': sharedSecret};
    if (publicKey != null) keys['publicKey'] = publicKey;
    if (serverPublicKey != null) keys['serverPublicKey'] = serverPublicKey;
    savedKeys[serverId] = keys;
  }

  @override
  Future<void> saveDeviceToken({
    required String serverId,
    required String deviceToken,
  }) async {
    savedTokens[serverId] = deviceToken;
  }

  @override
  Future<Uint8List?> getSharedSecret(String serverId) async {
    return savedKeys[serverId]?['sharedSecret'];
  }

  @override
  Future<String?> getDeviceToken(String serverId) async {
    return savedTokens[serverId];
  }

  @override
  Future<
    ({
      Uint8List? sharedSecret,
      Uint8List? publicKey,
      Uint8List? serverPublicKey,
    })
  >
  getServerKeys(String serverId) async {
    final keys = savedKeys[serverId];
    return (
      sharedSecret: keys?['sharedSecret'],
      publicKey: keys?['publicKey'],
      serverPublicKey: keys?['serverPublicKey'],
    );
  }

  @override
  Future<void> deleteServerData(String serverId) async {
    savedKeys.remove(serverId);
    savedTokens.remove(serverId);
  }
}

/// Fake [SettingsRepository] that records servers added to it.
class _FakeLocalSettingsRepository implements SettingsRepository {
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

/// Creates a [WsPairingRepository] wired to the given fakes.
///
/// The [connectionFactory] and [clientFactory] return the fakes so no real
/// WebSocket connection is ever made.
WsPairingRepository _createRepo({
  required _FakeWsBridgeClient fakeClient,
  required _FakeLocalSettingsRepository fakeSettingsRepo,
  required _FakeKeyExchange fakeKeyExchange,
  _FakeWsConnection? fakeConnection,
  _FakeSecureKeyStorage? fakeSecureKeyStorage,
}) {
  final conn = fakeConnection ?? _FakeWsConnection();
  return WsPairingRepository(
    settingsRepo: fakeSettingsRepo,
    keyExchange: fakeKeyExchange,
    secureKeyStorage: fakeSecureKeyStorage ?? _FakeSecureKeyStorage(),
    connectionFactory:
        ({
          required String bridgeUrl,
          required String serverId,
          required String deviceId,
        }) => conn,
    clientFactory:
        ({required WsConnection connection, required String deviceId}) =>
            fakeClient,
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('WsPairingRepository', () {
    late _FakeWsBridgeClient fakeClient;
    late _FakeLocalSettingsRepository fakeSettingsRepo;
    late _FakeKeyExchange fakeKeyExchange;
    late _FakeWsConnection fakeConnection;
    late WsPairingRepository repo;

    setUp(() {
      fakeClient = _FakeWsBridgeClient();
      fakeSettingsRepo = _FakeLocalSettingsRepository();
      fakeKeyExchange = _FakeKeyExchange();
      fakeConnection = _FakeWsConnection();

      repo = _createRepo(
        fakeClient: fakeClient,
        fakeSettingsRepo: fakeSettingsRepo,
        fakeKeyExchange: fakeKeyExchange,
        fakeConnection: fakeConnection,
      );
    });

    group('pair() result mapping', () {
      test('returns server with correct fields from WS response', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: 'My Workstation',
          serverId: 'srv-789',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-789',
          pairingToken: '123456',
        );

        expect(server.id, 'srv-789');
        expect(server.name, 'My Workstation');
        expect(server.bridgeUrl, 'ws://bridge:8080');
        expect(server.deviceId, 'dev-123');
        expect(server.deviceToken, 'tok-456');
        expect(server.isOnline, isTrue);
      });

      test('falls back to "CLI Server" when serverName is empty', () async {
        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-123',
          deviceToken: 'tok-456',
          serverName: '',
          serverId: 'srv-789',
          serverPublicKey: null,
        );

        final server = await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-789',
          pairingToken: '000000',
        );

        expect(server.name, 'CLI Server');
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
          bridgeUrl: 'ws://bridge:8080',
          serverId: '',
          pairingToken: '000000',
        );

        expect(server.id, startsWith('srv-'));
        expect(server.id, isNot(''));
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
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-1',
          pairingToken: '999999',
        );

        // KeyExchange was invoked.
        expect(fakeKeyExchange.generateKeypairCalled, isTrue);
        expect(fakeKeyExchange.computeSharedSecretCalled, isTrue);

        // Server has encryption keys populated.
        expect(server.hasEncryption, isTrue);
        expect(server.sharedSecret, isNotNull);
        expect(server.sharedSecret!.length, 32);
        expect(server.sharedSecret, _FakeKeyExchange.fakeSharedSecretBytes);
        expect(server.publicKey, _FakeKeyExchange.fakePublicKeyBytes);
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
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-2',
          pairingToken: '000000',
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
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-3',
          pairingToken: '000000',
        );

        expect(fakeKeyExchange.computeSharedSecretCalled, isFalse);
        expect(server.hasEncryption, isFalse);
      });

      test(
        'uses serverPublicKey from argument when response has none',
        () async {
          final qrServerPubKey = Uint8List.fromList(
            List.generate(32, (i) => i + 80),
          );

          fakeClient.pairResultToReturn = PairResult(
            deviceId: 'dev-123',
            deviceToken: 'tok-456',
            serverName: 'QR Server',
            serverId: 'srv-4',
            serverPublicKey: null,
          );

          final server = await repo.pair(
            bridgeUrl: 'ws://bridge:8080',
            serverId: 'srv-4',
            pairingToken: '123456',
            serverPublicKey: qrServerPubKey,
          );

          expect(fakeKeyExchange.computeSharedSecretCalled, isTrue);
          expect(server.hasEncryption, isTrue);
          expect(server.serverPublicKey, qrServerPubKey);
        },
      );
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
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-saved',
          pairingToken: '111111',
        );

        expect(fakeSettingsRepo.addedServers, hasLength(1));
        final saved = fakeSettingsRepo.addedServers.first;
        expect(saved.id, server.id);
        expect(saved.name, 'Saved Server');
        expect(saved.deviceToken, 'tok-1');
      });
    });

    group('pair() passes correct args to WS client', () {
      test('sends pairing token and device name', () async {
        await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-1',
          pairingToken: '654321',
        );

        expect(fakeClient.pairCalls, hasLength(1));
        expect(fakeClient.pairCalls.first.token, '654321');
        expect(fakeClient.pairCalls.first.deviceName, 'Flutter Mobile');
      });

      test('sends mobile public key for key exchange', () async {
        await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-1',
          pairingToken: '123456',
        );

        expect(fakeClient.pairCalls, hasLength(1));
        final sentKey = fakeClient.pairCalls.first.mobilePublicKey;
        expect(sentKey, isNotNull);
        expect(sentKey, _FakeKeyExchange.fakePublicKeyBytes);
      });
    });

    group('pair() client lifecycle', () {
      test('disposes WS client after successful pair', () async {
        await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-1',
          pairingToken: '123456',
        );

        expect(fakeClient.disposeCalled, isTrue);
      });

      test('disposes WS client even when pair throws', () async {
        fakeClient.pairError = Exception('bad token');

        try {
          await repo.pair(
            bridgeUrl: 'ws://bridge:8080',
            serverId: 'srv-1',
            pairingToken: 'bad',
          );
        } on Exception catch (_) {
          // Expected.
        }

        expect(fakeClient.disposeCalled, isTrue);
      });

      test('disposes WS connection after successful pair', () async {
        await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-1',
          pairingToken: '123456',
        );

        expect(fakeConnection.disposeCalled, isTrue);
      });

      test('disposes WS connection even when pair throws', () async {
        fakeClient.pairError = Exception('network error');

        try {
          await repo.pair(
            bridgeUrl: 'ws://bridge:8080',
            serverId: 'srv-1',
            pairingToken: 'bad',
          );
        } on Exception catch (_) {
          // Expected.
        }

        expect(fakeConnection.disposeCalled, isTrue);
      });

      test('connects WS connection before pairing', () async {
        await repo.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-1',
          pairingToken: '123456',
        );

        expect(fakeConnection.connectCalled, isTrue);
      });
    });

    group('pair() error handling', () {
      test('propagates Exception from client', () async {
        fakeClient.pairError = Exception('invalid token');

        await expectLater(
          repo.pair(
            bridgeUrl: 'ws://bridge:8080',
            serverId: 'srv-1',
            pairingToken: 'bad',
          ),
          throwsA(isA<Exception>()),
        );
      });

      test('does not persist server when pair fails', () async {
        fakeClient.pairError = Exception('server down');

        try {
          await repo.pair(
            bridgeUrl: 'ws://bridge:8080',
            serverId: 'srv-1',
            pairingToken: '000000',
          );
        } on Exception catch (_) {
          // Expected.
        }

        expect(fakeSettingsRepo.addedServers, isEmpty);
      });
    });

    group('pair() secure key storage', () {
      test('saves keys to secure storage when encryption negotiated', () async {
        final serverPubKey = Uint8List.fromList(
          List.generate(32, (i) => i + 50),
        );
        final fakeSecureStorage = _FakeSecureKeyStorage();

        final repoWithStorage = _createRepo(
          fakeClient: fakeClient,
          fakeSettingsRepo: fakeSettingsRepo,
          fakeKeyExchange: fakeKeyExchange,
          fakeConnection: fakeConnection,
          fakeSecureKeyStorage: fakeSecureStorage,
        );

        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-1',
          deviceToken: 'tok-1',
          serverName: 'Secure',
          serverId: 'srv-secure',
          serverPublicKey: serverPubKey,
        );

        await repoWithStorage.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-secure',
          pairingToken: '123456',
        );

        expect(fakeSecureStorage.savedKeys, contains('srv-secure'));
        expect(fakeSecureStorage.savedTokens['srv-secure'], 'tok-1');
      });

      test('saves device token even without encryption', () async {
        final fakeSecureStorage = _FakeSecureKeyStorage();

        final repoWithStorage = _createRepo(
          fakeClient: fakeClient,
          fakeSettingsRepo: fakeSettingsRepo,
          fakeKeyExchange: fakeKeyExchange,
          fakeConnection: fakeConnection,
          fakeSecureKeyStorage: fakeSecureStorage,
        );

        fakeClient.pairResultToReturn = PairResult(
          deviceId: 'dev-1',
          deviceToken: 'tok-plain',
          serverName: 'Plain',
          serverId: 'srv-plain',
          serverPublicKey: null,
        );

        await repoWithStorage.pair(
          bridgeUrl: 'ws://bridge:8080',
          serverId: 'srv-plain',
          pairingToken: '000000',
        );

        expect(fakeSecureStorage.savedTokens['srv-plain'], 'tok-plain');
      });
    });

    group('PairingRepository interface', () {
      test('WsPairingRepository implements PairingRepository', () {
        expect(repo, isA<PairingRepository>());
      });
    });
  });
}

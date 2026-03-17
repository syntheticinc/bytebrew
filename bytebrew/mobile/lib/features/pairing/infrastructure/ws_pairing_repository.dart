import 'dart:typed_data';

import 'package:bytebrew_mobile/core/crypto/key_exchange.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/secure_key_storage.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:bytebrew_mobile/features/settings/domain/settings_repository.dart';

/// [PairingRepository] implementation that pairs with a CLI server via
/// WebSocket through the Bridge relay.
///
/// Creates a temporary WS connection to the bridge, performs the pair
/// handshake, optionally negotiates E2E encryption keys, persists the
/// paired server to local storage, and cleans up the connection.
class WsPairingRepository implements PairingRepository {
  WsPairingRepository({
    required SettingsRepository settingsRepo,
    required KeyExchange keyExchange,
    SecureKeyStorage? secureKeyStorage,
    WsConnection Function({
      required String bridgeUrl,
      required String serverId,
      required String deviceId,
    })?
    connectionFactory,
    WsBridgeClient Function({
      required WsConnection connection,
      required String deviceId,
    })?
    clientFactory,
  }) : _settingsRepo = settingsRepo,
       _keyExchange = keyExchange,
       _secureKeyStorage = secureKeyStorage ?? SecureKeyStorage(),
       _connectionFactory = connectionFactory,
       _clientFactory = clientFactory;

  final SettingsRepository _settingsRepo;
  final KeyExchange _keyExchange;
  final SecureKeyStorage _secureKeyStorage;
  final WsConnection Function({
    required String bridgeUrl,
    required String serverId,
    required String deviceId,
  })?
  _connectionFactory;
  final WsBridgeClient Function({
    required WsConnection connection,
    required String deviceId,
  })?
  _clientFactory;

  @override
  Future<Server> pair({
    required String bridgeUrl,
    required String serverId,
    required String pairingToken,
    Uint8List? serverPublicKey,
  }) async {
    // Generate keypair for key exchange.
    final keyPair = await _keyExchange.generateKeypair();
    final publicKeyBytes = await _keyExchange.extractPublicKeyBytes(keyPair);

    // Create temporary WS connection to bridge.
    final tempDeviceId = 'pairing-${DateTime.now().millisecondsSinceEpoch}';
    final wsConnection = _connectionFactory != null
        ? _connectionFactory(
            bridgeUrl: bridgeUrl,
            serverId: serverId,
            deviceId: tempDeviceId,
          )
        : WsConnection(
            bridgeUrl: bridgeUrl,
            serverId: serverId,
            deviceId: tempDeviceId,
          );

    final client = _clientFactory != null
        ? _clientFactory(connection: wsConnection, deviceId: tempDeviceId)
        : WsBridgeClient(connection: wsConnection, deviceId: tempDeviceId);

    try {
      await wsConnection.connect();

      final result = await client.pair(
        token: pairingToken,
        deviceName: 'Flutter Mobile',
        mobilePublicKey: publicKeyBytes,
      );

      // Compute shared secret if server provided its public key.
      Uint8List? sharedSecret;
      Uint8List? resultServerPublicKey;
      Uint8List? ourPublicKey;

      // Use server public key from pair response, or the one from QR code.
      final effectiveServerPubKey = result.serverPublicKey ?? serverPublicKey;

      if (effectiveServerPubKey != null && effectiveServerPubKey.length >= 32) {
        final secret = await _keyExchange.computeSharedSecretFromBytes(
          keyPair,
          effectiveServerPubKey,
        );
        sharedSecret = Uint8List.fromList(await secret.extractBytes());
        resultServerPublicKey = effectiveServerPubKey;
        ourPublicKey = publicKeyBytes;
      }

      // Build the paired server.
      final resultServerId = result.serverId.isNotEmpty
          ? result.serverId
          : serverId.isNotEmpty
          ? serverId
          : 'srv-${DateTime.now().millisecondsSinceEpoch}';

      final serverName = result.serverName.isNotEmpty
          ? result.serverName
          : 'CLI Server';

      final server = Server(
        id: resultServerId,
        name: serverName,
        bridgeUrl: bridgeUrl,
        isOnline: true,
        latencyMs: 0,
        pairedAt: DateTime.now(),
        deviceId: result.deviceId,
        deviceToken: result.deviceToken,
        sharedSecret: sharedSecret,
        publicKey: ourPublicKey,
        serverPublicKey: resultServerPublicKey,
      );

      await _settingsRepo.addServer(server);

      // Persist sensitive keys to secure storage.
      if (sharedSecret != null) {
        await _secureKeyStorage.saveServerKeys(
          serverId: server.id,
          sharedSecret: sharedSecret,
          publicKey: ourPublicKey,
          serverPublicKey: resultServerPublicKey,
        );
      }
      if (result.deviceToken.isNotEmpty) {
        await _secureKeyStorage.saveDeviceToken(
          serverId: server.id,
          deviceToken: result.deviceToken,
        );
      }

      return server;
    } finally {
      await client.dispose();
      await wsConnection.dispose();
    }
  }
}

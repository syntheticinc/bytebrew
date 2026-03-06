import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Stores encryption keys and device tokens in platform-secure storage.
///
/// Uses [FlutterSecureStorage] (Keychain on iOS, EncryptedSharedPreferences
/// on Android) to keep sensitive pairing data out of plain SharedPreferences.
class SecureKeyStorage {
  SecureKeyStorage([FlutterSecureStorage? storage])
    : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  static const _prefix = 'bytebrew_server_';

  String _key(String serverId, String field) => '$_prefix${serverId}_$field';

  /// Saves encryption keys for a paired server.
  Future<void> saveServerKeys({
    required String serverId,
    required Uint8List sharedSecret,
    Uint8List? publicKey,
    Uint8List? serverPublicKey,
  }) async {
    await _storage.write(
      key: _key(serverId, 'sharedSecret'),
      value: base64Encode(sharedSecret),
    );
    if (publicKey != null) {
      await _storage.write(
        key: _key(serverId, 'publicKey'),
        value: base64Encode(publicKey),
      );
    }
    if (serverPublicKey != null) {
      await _storage.write(
        key: _key(serverId, 'serverPublicKey'),
        value: base64Encode(serverPublicKey),
      );
    }
  }

  /// Saves the device token for a paired server.
  Future<void> saveDeviceToken({
    required String serverId,
    required String deviceToken,
  }) async {
    await _storage.write(
      key: _key(serverId, 'deviceToken'),
      value: deviceToken,
    );
  }

  /// Reads the shared secret for [serverId], or null if not stored.
  Future<Uint8List?> getSharedSecret(String serverId) async {
    final value = await _storage.read(key: _key(serverId, 'sharedSecret'));
    if (value == null) return null;
    return base64Decode(value);
  }

  /// Reads the device token for [serverId], or null if not stored.
  Future<String?> getDeviceToken(String serverId) async {
    return _storage.read(key: _key(serverId, 'deviceToken'));
  }

  /// Reads all stored keys for [serverId].
  Future<
    ({
      Uint8List? sharedSecret,
      Uint8List? publicKey,
      Uint8List? serverPublicKey,
    })
  >
  getServerKeys(String serverId) async {
    return (
      sharedSecret: await _readBytes(serverId, 'sharedSecret'),
      publicKey: await _readBytes(serverId, 'publicKey'),
      serverPublicKey: await _readBytes(serverId, 'serverPublicKey'),
    );
  }

  /// Deletes all stored keys and tokens for [serverId].
  Future<void> deleteServerData(String serverId) async {
    for (final field in [
      'sharedSecret',
      'publicKey',
      'serverPublicKey',
      'deviceToken',
    ]) {
      await _storage.delete(key: _key(serverId, field));
    }
  }

  Future<Uint8List?> _readBytes(String serverId, String field) async {
    final value = await _storage.read(key: _key(serverId, field));
    if (value == null) return null;
    return base64Decode(value);
  }
}

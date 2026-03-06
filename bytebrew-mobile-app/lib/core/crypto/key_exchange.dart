import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

/// X25519 key exchange for E2E encryption pairing.
///
/// Wraps the `cryptography` package's [X25519] algorithm, providing
/// convenience methods for keypair generation, public-key extraction,
/// and shared-secret computation from both typed keys and raw bytes.
class KeyExchange {
  final _algorithm = X25519();

  /// Generates a new ephemeral X25519 keypair.
  Future<SimpleKeyPair> generateKeypair() => _algorithm.newKeyPair();

  /// Extracts the raw 32-byte public key from [keyPair].
  Future<Uint8List> extractPublicKeyBytes(SimpleKeyPair keyPair) async {
    final publicKey = await keyPair.extractPublicKey();
    final bytes = publicKey.bytes;
    return Uint8List.fromList(bytes);
  }

  /// Computes a shared secret from [ours] keypair and [theirs] public key.
  Future<SecretKey> computeSharedSecret(
    SimpleKeyPair ours,
    SimplePublicKey theirs,
  ) async {
    return _algorithm.sharedSecretKey(keyPair: ours, remotePublicKey: theirs);
  }

  /// Computes a shared secret from [ourKeyPair] and raw
  /// [theirPublicKeyBytes] (32 bytes).
  Future<SecretKey> computeSharedSecretFromBytes(
    SimpleKeyPair ourKeyPair,
    Uint8List theirPublicKeyBytes,
  ) async {
    final theirPublicKey = SimplePublicKey(
      theirPublicKeyBytes,
      type: KeyPairType.x25519,
    );
    return computeSharedSecret(ourKeyPair, theirPublicKey);
  }
}

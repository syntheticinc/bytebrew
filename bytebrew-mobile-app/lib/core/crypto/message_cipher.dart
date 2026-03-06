import 'dart:math';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

/// XChaCha20-Poly1305 message cipher compatible with Go's `crypto.go`.
///
/// Wire format: `nonce(24) || ciphertext+tag(16)`
/// Nonce layout: `random(16) || counter(8, little-endian)`
class MessageCipher {
  MessageCipher(this._sharedSecret);

  final Uint8List _sharedSecret;

  static const _nonceSize = 24;
  static const _randomPrefixSize = 16;
  static const _counterSize = 8;
  static const _tagSize = 16;

  final _algorithm = Xchacha20.poly1305Aead();
  final _random = Random.secure();

  /// Encrypts [plaintext] with [counter].
  ///
  /// Returns `nonce(24) || ciphertext+tag`.
  Future<Uint8List> encrypt(Uint8List plaintext, int counter) async {
    // Build nonce: 16 random bytes + 8 bytes counter (little-endian).
    final nonce = Uint8List(_nonceSize);
    for (var i = 0; i < _randomPrefixSize; i++) {
      nonce[i] = _random.nextInt(256);
    }
    _putUint64LE(nonce, _randomPrefixSize, counter);

    final secretKey = SecretKey(_sharedSecret);

    final secretBox = await _algorithm.encrypt(
      plaintext,
      secretKey: secretKey,
      nonce: nonce,
    );

    // Output: nonce || ciphertext || mac
    final result = Uint8List(
      _nonceSize + secretBox.cipherText.length + _tagSize,
    );
    result.setRange(0, _nonceSize, nonce);
    result.setRange(
      _nonceSize,
      _nonceSize + secretBox.cipherText.length,
      secretBox.cipherText,
    );
    result.setRange(
      _nonceSize + secretBox.cipherText.length,
      result.length,
      secretBox.mac.bytes,
    );

    return result;
  }

  /// Decrypts [data] in wire format `nonce(24) || ciphertext+tag`.
  ///
  /// Returns `(plaintext, counter)`.
  Future<(Uint8List, int)> decrypt(Uint8List data) async {
    if (data.length < _nonceSize + _tagSize) {
      throw ArgumentError(
        'ciphertext too short: need at least ${_nonceSize + _tagSize} bytes, '
        'got ${data.length}',
      );
    }

    final nonce = data.sublist(0, _nonceSize);
    final counter = _getUint64LE(nonce, _randomPrefixSize);

    final ciphertextWithTag = data.sublist(_nonceSize);
    final ciphertextLen = ciphertextWithTag.length - _tagSize;
    final ciphertext = ciphertextWithTag.sublist(0, ciphertextLen);
    final mac = Mac(ciphertextWithTag.sublist(ciphertextLen));

    final secretKey = SecretKey(_sharedSecret);

    final secretBox = SecretBox(ciphertext, nonce: nonce, mac: mac);

    final plaintext = await _algorithm.decrypt(secretBox, secretKey: secretKey);

    return (Uint8List.fromList(plaintext), counter);
  }

  /// Writes [value] as a little-endian uint64 into [buffer] at [offset].
  static void _putUint64LE(Uint8List buffer, int offset, int value) {
    for (var i = 0; i < _counterSize; i++) {
      buffer[offset + i] = (value >> (i * 8)) & 0xff;
    }
  }

  /// Reads a little-endian uint64 from [buffer] at [offset].
  static int _getUint64LE(Uint8List buffer, int offset) {
    var value = 0;
    for (var i = 0; i < _counterSize; i++) {
      value |= buffer[offset + i] << (i * 8);
    }
    return value;
  }
}

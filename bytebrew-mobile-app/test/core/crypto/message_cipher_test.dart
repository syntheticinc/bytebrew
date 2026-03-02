import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/crypto/message_cipher.dart';

void main() {
  late MessageCipher cipher;

  /// Helper: generates a random 32-byte shared secret via X25519 ECDH
  /// (same way production code would obtain one).
  Future<SecretKey> generateSharedSecret() async {
    final x25519 = X25519();
    final alice = await x25519.newKeyPair();
    final bob = await x25519.newKeyPair();
    final bobPub = await bob.extractPublicKey();
    return x25519.sharedSecretKey(keyPair: alice, remotePublicKey: bobPub);
  }

  setUp(() {
    cipher = MessageCipher();
  });

  group('MessageCipher', () {
    test('encrypts and decrypts message roundtrip', () async {
      final sharedSecret = await generateSharedSecret();
      final plaintext = Uint8List.fromList(utf8.encode('Hello, ByteBrew!'));
      const counter = 42;

      final encrypted = await cipher.encrypt(sharedSecret, plaintext, counter);
      final (decrypted, decryptedCounter) = await cipher.decrypt(
        sharedSecret,
        encrypted,
      );

      expect(decrypted, plaintext);
      expect(decryptedCounter, counter);
    });

    test('different messages produce different ciphertexts', () async {
      final sharedSecret = await generateSharedSecret();
      final msg1 = Uint8List.fromList(utf8.encode('Message one'));
      final msg2 = Uint8List.fromList(utf8.encode('Message two'));

      final encrypted1 = await cipher.encrypt(sharedSecret, msg1, 1);
      final encrypted2 = await cipher.encrypt(sharedSecret, msg2, 2);

      // Ciphertexts should differ (different plaintext + different nonce).
      expect(encrypted1, isNot(equals(encrypted2)));
    });

    test('decryption with wrong key fails', () async {
      final correctKey = await generateSharedSecret();
      final wrongKey = await generateSharedSecret();
      final plaintext = Uint8List.fromList(utf8.encode('Secret data'));

      final encrypted = await cipher.encrypt(correctKey, plaintext, 1);

      // Decrypting with the wrong key should throw (Poly1305 auth failure).
      expect(
        () => cipher.decrypt(wrongKey, encrypted),
        throwsA(isA<Object>()),
      );
    });

    test('empty message roundtrip', () async {
      final sharedSecret = await generateSharedSecret();
      final emptyPlaintext = Uint8List(0);
      const counter = 0;

      final encrypted = await cipher.encrypt(
        sharedSecret,
        emptyPlaintext,
        counter,
      );

      // Wire format: nonce(24) + ciphertext(0) + tag(16) = 40 bytes.
      expect(encrypted.length, 40);

      final (decrypted, decryptedCounter) = await cipher.decrypt(
        sharedSecret,
        encrypted,
      );

      expect(decrypted, isEmpty);
      expect(decryptedCounter, counter);
    });

    test('large message roundtrip', () async {
      final sharedSecret = await generateSharedSecret();
      // 10 KB of data.
      final largePlaintext = Uint8List(10 * 1024);
      for (var i = 0; i < largePlaintext.length; i++) {
        largePlaintext[i] = i % 256;
      }
      const counter = 999;

      final encrypted = await cipher.encrypt(
        sharedSecret,
        largePlaintext,
        counter,
      );
      final (decrypted, decryptedCounter) = await cipher.decrypt(
        sharedSecret,
        encrypted,
      );

      expect(decrypted, largePlaintext);
      expect(decryptedCounter, counter);
    });

    test('decrypt throws on data too short', () async {
      final sharedSecret = await generateSharedSecret();
      // Minimum is 40 bytes (nonce 24 + tag 16). Provide less.
      final tooShort = Uint8List(39);

      expect(
        () => cipher.decrypt(sharedSecret, tooShort),
        throwsA(isA<ArgumentError>()),
      );
    });

    test('counter is preserved in nonce', () async {
      final sharedSecret = await generateSharedSecret();
      final plaintext = Uint8List.fromList(utf8.encode('counter test'));

      // Use a large counter value to test uint64 encoding.
      const largeCounter = 0x00FFFFFFFFFFFFFF;

      final encrypted = await cipher.encrypt(
        sharedSecret,
        plaintext,
        largeCounter,
      );
      final (_, decryptedCounter) = await cipher.decrypt(
        sharedSecret,
        encrypted,
      );

      expect(decryptedCounter, largeCounter);
    });

    test('same message encrypted twice produces different ciphertexts', () async {
      final sharedSecret = await generateSharedSecret();
      final plaintext = Uint8List.fromList(utf8.encode('same message'));

      final encrypted1 = await cipher.encrypt(sharedSecret, plaintext, 1);
      final encrypted2 = await cipher.encrypt(sharedSecret, plaintext, 1);

      // Even with the same counter, the random 16 bytes in the nonce
      // should make ciphertexts differ.
      expect(encrypted1, isNot(equals(encrypted2)));
    });

    test('wire format has correct structure', () async {
      final sharedSecret = await generateSharedSecret();
      final plaintext = Uint8List.fromList(utf8.encode('format test'));
      const counter = 7;

      final encrypted = await cipher.encrypt(
        sharedSecret,
        plaintext,
        counter,
      );

      // Wire format: nonce(24) + ciphertext(N) + tag(16)
      // N == plaintext.length for stream ciphers.
      final expectedLength = 24 + plaintext.length + 16;
      expect(encrypted.length, expectedLength);

      // Verify counter is embedded in nonce bytes [16..24] as little-endian.
      final counterBytes = ByteData.sublistView(
        Uint8List.fromList(encrypted.sublist(16, 24)),
      );
      final extractedCounter = counterBytes.getUint64(0, Endian.little);
      expect(extractedCounter, counter);
    });
  });
}

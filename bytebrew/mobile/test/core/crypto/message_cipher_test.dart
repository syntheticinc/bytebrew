import 'dart:typed_data';

import 'package:bytebrew_mobile/core/crypto/message_cipher.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MessageCipher', () {
    late Uint8List sharedSecret;

    setUp(() async {
      // Generate a real shared secret via X25519 key exchange.
      final algorithm = X25519();
      final kp1 = await algorithm.newKeyPair();
      final kp2 = await algorithm.newKeyPair();
      final pub2 = await kp2.extractPublicKey();
      final secret = await algorithm.sharedSecretKey(
        keyPair: kp1,
        remotePublicKey: pub2,
      );
      sharedSecret = Uint8List.fromList(await secret.extractBytes());
    });

    test(
      'round trip: encrypt then decrypt returns original plaintext',
      () async {
        final cipher = MessageCipher(sharedSecret);
        final plaintext = Uint8List.fromList(
          'hello, encrypted world!'.codeUnits,
        );
        const counter = 42;

        final encrypted = await cipher.encrypt(plaintext, counter);
        final (decrypted, gotCounter) = await cipher.decrypt(encrypted);

        expect(decrypted, equals(plaintext));
        expect(gotCounter, equals(counter));
      },
    );

    test('empty plaintext', () async {
      final cipher = MessageCipher(sharedSecret);
      final plaintext = Uint8List(0);

      final encrypted = await cipher.encrypt(plaintext, 0);
      final (decrypted, counter) = await cipher.decrypt(encrypted);

      expect(decrypted, isEmpty);
      expect(counter, equals(0));
    });

    test('counter is preserved', () async {
      final cipher = MessageCipher(sharedSecret);
      final plaintext = Uint8List.fromList([1, 2, 3]);
      // Use a large counter value (within safe JS integer range).
      const largeCounter = 9007199254740991; // 2^53 - 1

      final encrypted = await cipher.encrypt(plaintext, largeCounter);
      final (_, gotCounter) = await cipher.decrypt(encrypted);

      expect(gotCounter, equals(largeCounter));
    });

    test('wire format: nonce(24) || ciphertext+tag', () async {
      final cipher = MessageCipher(sharedSecret);
      final plaintext = Uint8List.fromList([10, 20, 30]);

      final encrypted = await cipher.encrypt(plaintext, 1);

      // At least 24 (nonce) + 16 (tag) + 3 (plaintext) = 43 bytes.
      expect(encrypted.length, greaterThanOrEqualTo(43));

      // First 24 bytes = nonce.
      final nonce = encrypted.sublist(0, 24);
      expect(nonce.length, equals(24));
    });

    test('different keys cannot decrypt', () async {
      final cipher1 = MessageCipher(sharedSecret);

      // Generate a different shared secret.
      final algorithm = X25519();
      final kp3 = await algorithm.newKeyPair();
      final kp4 = await algorithm.newKeyPair();
      final pub4 = await kp4.extractPublicKey();
      final secret2 = await algorithm.sharedSecretKey(
        keyPair: kp3,
        remotePublicKey: pub4,
      );
      final otherSecret = Uint8List.fromList(await secret2.extractBytes());
      final cipher2 = MessageCipher(otherSecret);

      final plaintext = Uint8List.fromList('secret message'.codeUnits);
      final encrypted = await cipher1.encrypt(plaintext, 1);

      // Decrypt with wrong key should throw.
      expect(() => cipher2.decrypt(encrypted), throwsA(anything));
    });

    test('tampered ciphertext fails', () async {
      final cipher = MessageCipher(sharedSecret);
      final plaintext = Uint8List.fromList('important data'.codeUnits);

      final encrypted = await cipher.encrypt(plaintext, 1);

      // Flip a byte in the ciphertext portion (after nonce).
      final tampered = Uint8List.fromList(encrypted);
      tampered[25] ^= 0xff;

      expect(() => cipher.decrypt(tampered), throwsA(anything));
    });

    test('truncated data fails', () async {
      final cipher = MessageCipher(sharedSecret);

      expect(
        () => cipher.decrypt(Uint8List(10)),
        throwsA(isA<ArgumentError>()),
      );
    });

    test(
      'each encryption produces different ciphertext (random nonce)',
      () async {
        final cipher = MessageCipher(sharedSecret);
        final plaintext = Uint8List.fromList([1, 2, 3, 4, 5]);

        final encrypted1 = await cipher.encrypt(plaintext, 1);
        final encrypted2 = await cipher.encrypt(plaintext, 1);

        // Same plaintext and counter, but random nonce prefix differs.
        expect(encrypted1, isNot(equals(encrypted2)));
      },
    );

    test('both sides compute same result (ECDH symmetry)', () async {
      // Simulate two parties: mobile and server.
      final algorithm = X25519();
      final mobileKP = await algorithm.newKeyPair();
      final serverKP = await algorithm.newKeyPair();
      final mobilePub = await mobileKP.extractPublicKey();
      final serverPub = await serverKP.extractPublicKey();

      // Mobile computes shared secret.
      final mobileSecret = await algorithm.sharedSecretKey(
        keyPair: mobileKP,
        remotePublicKey: serverPub,
      );
      final mobileSecretBytes = Uint8List.fromList(
        await mobileSecret.extractBytes(),
      );

      // Server computes shared secret.
      final serverSecret = await algorithm.sharedSecretKey(
        keyPair: serverKP,
        remotePublicKey: mobilePub,
      );
      final serverSecretBytes = Uint8List.fromList(
        await serverSecret.extractBytes(),
      );

      // Both should have the same shared secret.
      expect(mobileSecretBytes, equals(serverSecretBytes));

      // Mobile encrypts, server decrypts.
      final mobileCipher = MessageCipher(mobileSecretBytes);
      final serverCipher = MessageCipher(serverSecretBytes);

      final plaintext = Uint8List.fromList('cross-party message'.codeUnits);
      final encrypted = await mobileCipher.encrypt(plaintext, 99);
      final (decrypted, counter) = await serverCipher.decrypt(encrypted);

      expect(decrypted, equals(plaintext));
      expect(counter, equals(99));
    });
  });
}

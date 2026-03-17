import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/crypto/key_exchange.dart';

void main() {
  late KeyExchange keyExchange;

  setUp(() {
    keyExchange = KeyExchange();
  });

  group('KeyExchange', () {
    test('generates keypair with correct length', () async {
      final keyPair = await keyExchange.generateKeypair();

      final publicKeyBytes = await keyExchange.extractPublicKeyBytes(keyPair);
      final privateKey = await keyPair.extractPrivateKeyBytes();

      expect(publicKeyBytes, isNotEmpty);
      expect(privateKey, isNotEmpty);
      // X25519 keys are 32 bytes.
      expect(publicKeyBytes.length, 32);
      expect(privateKey.length, 32);
    });

    test('computes shared secret from keypair exchange', () async {
      // Alice and Bob each generate a keypair.
      final aliceKeyPair = await keyExchange.generateKeypair();
      final bobKeyPair = await keyExchange.generateKeypair();

      final alicePublicKey = await aliceKeyPair.extractPublicKey();
      final bobPublicKey = await bobKeyPair.extractPublicKey();

      // Each side computes the shared secret using their private key
      // and the other side's public key.
      final aliceSharedSecret = await keyExchange.computeSharedSecret(
        aliceKeyPair,
        bobPublicKey,
      );
      final bobSharedSecret = await keyExchange.computeSharedSecret(
        bobKeyPair,
        alicePublicKey,
      );

      final aliceSecretBytes = await aliceSharedSecret.extractBytes();
      final bobSecretBytes = await bobSharedSecret.extractBytes();

      // ECDH: both sides should arrive at the same shared secret.
      expect(aliceSecretBytes, bobSecretBytes);
      expect(aliceSecretBytes.length, 32);
    });

    test('different keypairs produce different shared secrets', () async {
      final alice = await keyExchange.generateKeypair();
      final bob1 = await keyExchange.generateKeypair();
      final bob2 = await keyExchange.generateKeypair();

      final bob1PublicKey = await bob1.extractPublicKey();
      final bob2PublicKey = await bob2.extractPublicKey();

      final secret1 = await keyExchange.computeSharedSecret(
        alice,
        bob1PublicKey,
      );
      final secret2 = await keyExchange.computeSharedSecret(
        alice,
        bob2PublicKey,
      );

      final secretBytes1 = await secret1.extractBytes();
      final secretBytes2 = await secret2.extractBytes();

      expect(secretBytes1, isNot(equals(secretBytes2)));
    });

    test('shared secret is deterministic', () async {
      final alice = await keyExchange.generateKeypair();
      final bob = await keyExchange.generateKeypair();

      final bobPublicKey = await bob.extractPublicKey();

      // Compute the shared secret twice with the same inputs.
      final secret1 = await keyExchange.computeSharedSecret(
        alice,
        bobPublicKey,
      );
      final secret2 = await keyExchange.computeSharedSecret(
        alice,
        bobPublicKey,
      );

      final secretBytes1 = await secret1.extractBytes();
      final secretBytes2 = await secret2.extractBytes();

      expect(secretBytes1, secretBytes2);
    });

    test('computeSharedSecretFromBytes works with raw byte arrays', () async {
      final alice = await keyExchange.generateKeypair();
      final bob = await keyExchange.generateKeypair();

      final bobPublicKeyBytes = await keyExchange.extractPublicKeyBytes(bob);

      // Use the convenience method that accepts raw bytes.
      final secret = await keyExchange.computeSharedSecretFromBytes(
        alice,
        bobPublicKeyBytes,
      );

      // Compare with the result from the regular method.
      final bobPublicKey = SimplePublicKey(
        bobPublicKeyBytes,
        type: KeyPairType.x25519,
      );
      final secretDirect = await keyExchange.computeSharedSecret(
        alice,
        bobPublicKey,
      );

      final secretBytes = await secret.extractBytes();
      final secretDirectBytes = await secretDirect.extractBytes();

      expect(secretBytes, secretDirectBytes);
    });
  });
}

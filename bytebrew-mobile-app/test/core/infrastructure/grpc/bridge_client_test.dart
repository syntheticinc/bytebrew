import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/infrastructure/grpc/bridge_client.dart';

void main() {
  group('BridgeClient', () {
    test('initial state is not connected', () {
      final client = BridgeClient(bridgeUrl: 'localhost:8443');

      expect(client.isConnected, isFalse);
    });

    test('close without connect does not throw', () async {
      final client = BridgeClient(bridgeUrl: 'localhost:8443');

      // Closing a client that was never connected should not throw.
      await expectLater(client.close(), completes);
    });

    test('sendFrame when not connected does not throw', () {
      final client = BridgeClient(bridgeUrl: 'localhost:8443');
      final payload = Uint8List.fromList([1, 2, 3]);

      // sendFrame on a disconnected client should silently return
      // (debugPrint a warning, but no exception).
      expect(() => client.sendFrame(payload), returnsNormally);
    });

    test('isConnected remains false after close', () async {
      final client = BridgeClient(bridgeUrl: 'localhost:8443');

      await client.close();

      expect(client.isConnected, isFalse);
    });

    test('incomingFrames stream is available before connect', () {
      final client = BridgeClient(bridgeUrl: 'localhost:8443');

      // The incoming frames stream should be accessible even before
      // connect() is called (it is a broadcast stream controller).
      expect(client.incomingFrames, isA<Stream<Uint8List>>());
    });

    test('multiple close calls do not throw', () async {
      final client = BridgeClient(bridgeUrl: 'localhost:8443');

      // Calling close multiple times should be safe.
      await expectLater(client.close(), completes);
      await expectLater(client.close(), completes);
    });
  });
}

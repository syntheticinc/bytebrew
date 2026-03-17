import 'dart:typed_data';

import 'package:bytebrew_mobile/core/domain/server.dart';

/// Repository for server pairing operations.
abstract class PairingRepository {
  /// Pair with a CLI server via Bridge relay.
  ///
  /// [bridgeUrl] is the Bridge relay URL (e.g. "ws://bridge.example.com:8080").
  /// [serverId] is the CLI server's unique ID for Bridge routing.
  /// [pairingToken] is the pairing token from the QR code or manual entry.
  /// [serverPublicKey] is the server's X25519 public key from the QR code
  /// (optional, may also come from the pair response).
  Future<Server> pair({
    required String bridgeUrl,
    required String serverId,
    required String pairingToken,
    Uint8List? serverPublicKey,
  });
}

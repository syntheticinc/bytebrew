import 'package:bytebrew_mobile/core/domain/server.dart';

/// Repository for server pairing operations.
abstract class PairingRepository {
  /// Pair with a server using its LAN address and pairing code.
  ///
  /// [serverAddress] is the server's host (e.g. "192.168.1.5").
  /// [pairingCode] is the 6-digit code displayed by `bytebrew mobile-pair`.
  Future<Server> pair({
    required String serverAddress,
    required String pairingCode,
  });
}

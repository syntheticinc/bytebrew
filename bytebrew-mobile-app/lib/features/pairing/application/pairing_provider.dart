import 'dart:async';

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:bytebrew_mobile/features/pairing/infrastructure/lan_pairing_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'pairing_provider.g.dart';

/// Provides the [PairingRepository] implementation backed by LAN WebSocket.
@Riverpod(keepAlive: true)
PairingRepository pairingRepository(Ref ref) {
  return LanPairingRepository();
}

/// Manages server pairing state.
@Riverpod(keepAlive: true)
class PairDevice extends _$PairDevice {
  @override
  FutureOr<Server?> build() => null;

  /// Initiates pairing with the given server address and pairing code.
  Future<Server> pair({
    required String serverAddress,
    required String pairingCode,
  }) async {
    state = const AsyncLoading();
    final repo = ref.read(pairingRepositoryProvider);
    final server = await repo.pair(
      serverAddress: serverAddress,
      pairingCode: pairingCode,
    );
    state = AsyncData(server);
    return server;
  }
}

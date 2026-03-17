import 'dart:async';
import 'dart:typed_data';

import 'package:bytebrew_mobile/core/crypto/key_exchange.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:bytebrew_mobile/features/pairing/infrastructure/ws_pairing_repository.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'pairing_provider.g.dart';

/// Provides the [PairingRepository] implementation backed by WebSocket.
@Riverpod(keepAlive: true)
PairingRepository pairingRepository(Ref ref) {
  final settingsRepo = ref.read(settingsRepositoryProvider);
  return WsPairingRepository(
    settingsRepo: settingsRepo,
    keyExchange: KeyExchange(),
  );
}

/// Manages server pairing state.
@Riverpod(keepAlive: true)
class PairDevice extends _$PairDevice {
  @override
  FutureOr<Server?> build() => null;

  /// Initiates pairing via Bridge with the given parameters.
  Future<Server> pair({
    required String bridgeUrl,
    required String serverId,
    required String pairingToken,
    Uint8List? serverPublicKey,
  }) async {
    state = const AsyncLoading();
    final repo = ref.read(pairingRepositoryProvider);
    final server = await repo.pair(
      bridgeUrl: bridgeUrl,
      serverId: serverId,
      pairingToken: pairingToken,
      serverPublicKey: serverPublicKey,
    );
    state = AsyncData(server);
    return server;
  }
}

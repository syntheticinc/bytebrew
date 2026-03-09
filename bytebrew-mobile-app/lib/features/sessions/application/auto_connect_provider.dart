import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auto_connect_provider.g.dart';

/// Automatically connects to all saved servers when first read.
///
/// This provider is kept alive so the connections persist for the app's
/// lifetime. It should be watched from the sessions screen to trigger
/// on first navigation.
///
/// On first run: calls connectToAll which initiates WS connections.
/// After that: periodic check every 30s to reconnect any failed servers.
@Riverpod(keepAlive: true)
Future<void> sessionsAutoConnect(Ref ref) async {
  bool disposed = false;
  ref.onDispose(() => disposed = true);

  final manager = ref.read(connectionManagerProvider);
  final repo = ref.read(settingsRepositoryProvider);
  final servers = await repo.getServersWithKeys();
  print('[AutoConnect] servers=${servers.length}');
  for (final s in servers) {
    print('[AutoConnect] server=${s.id} name=${s.name} deviceToken=${s.deviceToken != null ? '${s.deviceToken!.substring(0, 8)}...' : 'NULL'} bridgeUrl=${s.bridgeUrl}');
  }
  if (servers.isEmpty) return;

  // Initial connect — completes the Future so sessionsProvider can proceed.
  await manager.connectToAll(servers);

  // Periodically reconnect in background without blocking the Future.
  // Uses a Completer per sleep so dispose can cancel the pending timer.
  Future<void> cancellableSleep(Duration duration) async {
    final completer = Completer<void>();
    final timer = Timer(duration, completer.complete);
    ref.onDispose(timer.cancel);
    return completer.future;
  }

  // ignore: unawaited_futures
  () async {
    await cancellableSleep(const Duration(seconds: 10));
    while (!disposed) {
      // Reconnect servers that are disconnected (intentional close) or
      // stuck in error (e.g. initial ping failed, WsConnection was disconnected).
      // WsConnection handles socket-level reconnect internally for transient errors.
      final needsReconnect = manager.connections.values.any(
        (c) => c.status == WsConnectionStatus.disconnected ||
               c.status == WsConnectionStatus.error,
      );
      if (needsReconnect) {
        final current = await repo.getServersWithKeys();
        await manager.connectToAll(current);
      }
      await cancellableSleep(const Duration(seconds: 30));
    }
  }();
}

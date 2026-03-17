import 'dart:async';

import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:bytebrew_mobile/features/settings/infrastructure/local_settings_repository.dart';
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
  if (servers.isEmpty) return;

  // Initial connect — completes the Future so sessionsProvider can proceed.
  await manager.connectToAll(servers);

  // Update persisted server names if the server hostname changed since pairing.
  await _syncServerNames(ref, manager);

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

/// Checks connected servers for hostname changes and updates persistence.
Future<void> _syncServerNames(Ref ref, WsConnectionManager manager) async {
  final repo =
      ref.read(settingsRepositoryProvider) as LocalSettingsRepository;
  bool anyUpdated = false;

  for (final conn in manager.connections.values) {
    final newName = conn.resolvedName;
    if (newName == null || newName.isEmpty) continue;
    if (newName == conn.server.name) continue;

    // Server hostname changed — update in SharedPreferences.
    final updated = conn.server.copyWith(name: newName);
    await repo.addServer(updated);
    anyUpdated = true;
  }

  if (anyUpdated) {
    ref.invalidate(serversProvider);
  }
}

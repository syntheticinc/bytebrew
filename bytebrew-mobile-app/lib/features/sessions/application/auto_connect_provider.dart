import 'dart:async';

import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auto_connect_provider.g.dart';

/// Automatically connects to the first saved server when first read.
///
/// This provider is kept alive so the connection persists for the app's
/// lifetime. It should be watched from the sessions screen to trigger
/// on first navigation.
@Riverpod(keepAlive: true)
Future<void> sessionsAutoConnect(Ref ref) async {
  final notifier = ref.read(wsConnectionProvider.notifier);
  final repo = ref.read(settingsRepositoryProvider);
  final servers = repo.getServers();

  if (servers.isEmpty) return;

  // Connect to the first server.
  final server = servers.first;
  await notifier.connect(server.wsUrl);

  // Force session providers to re-evaluate now that the connection is ready.
  ref.invalidate(sessionRepositoryProvider);
}

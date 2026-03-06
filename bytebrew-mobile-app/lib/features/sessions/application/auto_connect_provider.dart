import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auto_connect_provider.g.dart';

/// Automatically connects to all saved servers when first read.
///
/// This provider is kept alive so the connections persist for the app's
/// lifetime. It should be watched from the sessions screen to trigger
/// on first navigation.
@Riverpod(keepAlive: true)
Future<void> sessionsAutoConnect(Ref ref) async {
  final manager = ref.read(connectionManagerProvider);
  final repo = ref.read(settingsRepositoryProvider);
  final servers = await repo.getServersWithKeys();
  if (servers.isEmpty) return;
  await manager.connectToAll(servers);
}

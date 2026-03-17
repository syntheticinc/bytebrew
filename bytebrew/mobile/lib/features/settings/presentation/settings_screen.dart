import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/domain/server.dart';
import '../../../core/infrastructure/ws/ws_connection.dart';
import '../../../core/infrastructure/ws/ws_connection_manager.dart';
import '../../../core/infrastructure/ws/ws_providers.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/widgets/status_indicator.dart';
import '../application/settings_provider.dart';
import '../infrastructure/local_settings_repository.dart';
import 'widgets/appearance_section.dart';
import 'widgets/notification_toggles.dart';
import 'widgets/server_list_tile.dart';

/// Full settings screen with branded styling.
class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final servers = ref.watch(serversProvider);
    final manager = ref.watch(connectionManagerProvider);
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          const _SectionHeader(label: 'SERVERS'),
          if (servers.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Text(
                'No servers paired yet',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: AppColors.shade3,
                ),
              ),
            ),
          ...servers.map(
            (server) => ServerListTile(
              server: server,
              onDismissed: () => _removeServer(ref, server.id),
            ),
          ),
          ListTile(
            leading: const Icon(Icons.add),
            title: const Text('Add Server'),
            onTap: () => context.push('/add-server'),
          ),
          const SizedBox(height: 16),
          const _SectionHeader(label: 'CONNECTION'),
          _BridgeStatusTile(manager: manager, servers: servers),
          const SizedBox(height: 16),
          const _SectionHeader(label: 'NOTIFICATIONS'),
          const NotificationToggles(),
          const SizedBox(height: 16),
          const _SectionHeader(label: 'APPEARANCE'),
          const AppearanceSection(),
          const SizedBox(height: 32),
          Center(
            child: Text(
              'About | Privacy | v0.1.0',
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
            ),
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }

  Future<void> _removeServer(WidgetRef ref, String serverId) async {
    // Remove from persistence first, then invalidate the server list.
    // Disconnect AFTER the UI rebuild so the Dismissible animation completes
    // without a conflicting rebuild from connectionManagerProvider notification.
    final repo =
        ref.read(settingsRepositoryProvider) as LocalSettingsRepository;
    await repo.removeServer(serverId);
    ref.invalidate(serversProvider);

    // Disconnect in the next frame — after the widget tree has rebuilt
    // without the removed server.
    final manager = ref.read(connectionManagerProvider);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      manager.disconnectFromServer(serverId);
    });
  }
}

/// Uppercase section header with monospace letterSpacing.
class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: AppColors.shade3,
          fontWeight: FontWeight.w600,
          letterSpacing: 2,
        ),
      ),
    );
  }
}

/// Single-line tile showing bridge relay connection status.
///
/// Displays "Bridge: host (Connected)" when any server is connected,
/// or prompts setup when no servers are paired.
class _BridgeStatusTile extends ConsumerWidget {
  const _BridgeStatusTile({required this.manager, required this.servers});

  final WsConnectionManager manager;
  final List<Server> servers;

  Future<void> _onReconnect(WidgetRef ref) async {
    final repo = ref.read(settingsRepositoryProvider) as LocalSettingsRepository;
    final serversWithKeys = await repo.getServersWithKeys();
    for (final server in serversWithKeys) {
      await manager.disconnectFromServer(server.id);
      await manager.connectToServer(server);
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (servers.isEmpty) {
      return const ListTile(
        leading: Icon(Icons.cloud_off, color: AppColors.shade3),
        title: Text('Bridge'),
        subtitle: Text('No servers paired'),
      );
    }

    final hasConnected = manager.activeConnections.isNotEmpty;
    final hasConnecting = manager.connections.values
        .any((c) => c.status == WsConnectionStatus.connecting);

    final Color statusColor;
    final String statusLabel;
    if (hasConnected) {
      statusColor = AppColors.statusActive;
      statusLabel = 'Connected';
    } else if (hasConnecting) {
      statusColor = AppColors.statusNeedsAttention;
      statusLabel = 'Connecting...';
    } else {
      statusColor = AppColors.shade3;
      statusLabel = 'Disconnected';
    }

    // Extract host from first server's bridge URL.
    final bridgeHost = _extractHost(servers.first.bridgeUrl);

    return ListTile(
      leading: StatusIndicator(color: statusColor),
      title: Text('Bridge: $bridgeHost'),
      subtitle: Text(statusLabel),
      onTap: () => _onReconnect(ref),
    );
  }

  String _extractHost(String url) {
    final cleaned = url
        .replaceFirst('wss://', '')
        .replaceFirst('ws://', '');
    final slashIndex = cleaned.indexOf('/');
    if (slashIndex >= 0) {
      return cleaned.substring(0, slashIndex);
    }
    return cleaned;
  }
}

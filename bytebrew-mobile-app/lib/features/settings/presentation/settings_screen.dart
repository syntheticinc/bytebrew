import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

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
    final isDark = theme.brightness == Brightness.dark;

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
          const _SectionHeader(label: 'BRIDGE'),
          _ConnectionCard(theme: theme, isDark: isDark, manager: manager),
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
    final manager = ref.read(connectionManagerProvider);
    await manager.disconnectFromServer(serverId);

    final repo =
        ref.read(settingsRepositoryProvider) as LocalSettingsRepository;
    await repo.removeServer(serverId);

    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.invalidate(serversProvider);
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

/// Card showing bridge relay connection status.
class _ConnectionCard extends StatelessWidget {
  const _ConnectionCard({
    required this.theme,
    required this.isDark,
    required this.manager,
  });

  final ThemeData theme;
  final bool isDark;
  final WsConnectionManager manager;

  @override
  Widget build(BuildContext context) {
    final connections = manager.connections;
    final connectedCount = manager.activeConnections.length;
    final totalCount = connections.length;

    final hasConnections = totalCount > 0;
    final allConnected = hasConnections && connectedCount == totalCount;

    final statusColor = allConnected
        ? AppColors.statusActive
        : AppColors.shade3;
    final statusText = !hasConnections
        ? 'No servers'
        : '$connectedCount / $totalCount connected';

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: isDark ? AppColors.darkAlt : AppColors.white,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: AppColors.shade3.withValues(alpha: 0.15)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.cloud, color: AppColors.shade3),
                const SizedBox(width: 8),
                Text('Bridge Relay', style: theme.textTheme.titleSmall),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                StatusIndicator(color: statusColor),
                const SizedBox(width: 8),
                Text(
                  statusText,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: statusColor,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              'Connects to your CLI servers via bridge relay',
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

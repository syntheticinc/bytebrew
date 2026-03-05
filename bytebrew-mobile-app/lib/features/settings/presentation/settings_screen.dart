import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/infrastructure/ws/ws_connection.dart';
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
    final wsStatus = ref.watch(wsConnectionProvider);
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
          const _SectionHeader(label: 'CONNECTION'),
          _ConnectionCard(theme: theme, isDark: isDark, wsStatus: wsStatus),
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
    // Disconnect if this is the connected server.
    final wsNotifier = ref.read(wsConnectionProvider.notifier);
    wsNotifier.disconnect();

    // Remove from persistent storage.
    final repo =
        ref.read(settingsRepositoryProvider) as LocalSettingsRepository;
    await repo.removeServer(serverId);

    // Invalidate the servers provider so the list rebuilds.
    ref.invalidate(serversProvider);
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

/// Card showing WebSocket connection status.
class _ConnectionCard extends StatelessWidget {
  const _ConnectionCard({
    required this.theme,
    required this.isDark,
    required this.wsStatus,
  });

  final ThemeData theme;
  final bool isDark;
  final WsConnectionStatus wsStatus;

  @override
  Widget build(BuildContext context) {
    final isConnected = wsStatus == WsConnectionStatus.connected;

    final statusColor = isConnected ? AppColors.statusActive : AppColors.shade3;
    final statusText = switch (wsStatus) {
      WsConnectionStatus.connected => 'Connected',
      WsConnectionStatus.connecting => 'Connecting...',
      WsConnectionStatus.error => 'Connection error',
      WsConnectionStatus.disconnected => 'Not connected',
    };

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
                Icon(Icons.wifi, color: AppColors.shade3),
                const SizedBox(width: 8),
                Text('WebSocket', style: theme.textTheme.titleSmall),
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
              'Connects to your CLI via WebSocket on the local network',
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

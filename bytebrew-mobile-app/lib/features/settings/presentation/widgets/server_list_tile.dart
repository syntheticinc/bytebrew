import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/domain/server.dart';
import '../../../../core/infrastructure/ws/ws_connection.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/status_indicator.dart';

/// A list tile displaying a paired server's name and connection status.
///
/// Reads live connection state from [WsConnection] to show online/offline status.
class ServerListTile extends ConsumerWidget {
  const ServerListTile({super.key, required this.server, this.onDismissed});

  final Server server;

  /// Called when the user swipes to dismiss the tile.
  final VoidCallback? onDismissed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Listen to WsConnection for live connection state.
    final wsStatus = ref.watch(wsConnectionProvider);
    final isConnected = wsStatus == WsConnectionStatus.connected;
    final statusColor = isConnected ? AppColors.statusActive : AppColors.shade3;

    final statusLabel = switch (wsStatus) {
      WsConnectionStatus.connected => 'Connected',
      WsConnectionStatus.connecting => 'Connecting',
      WsConnectionStatus.error => 'Error',
      WsConnectionStatus.disconnected => 'Offline',
    };

    final subtitle = '${server.lanAddress}:${server.wsPort}';

    final tile = ListTile(
      leading: StatusIndicator(color: statusColor),
      title: Text(server.name),
      subtitle: Text(subtitle),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [_StatusChip(label: statusLabel, isConnected: isConnected)],
      ),
    );

    if (onDismissed == null) return tile;

    return Dismissible(
      key: ValueKey(server.id),
      direction: DismissDirection.endToStart,
      background: Container(
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 24),
        color: AppColors.statusNeedsAttention,
        child: const Icon(Icons.delete, color: AppColors.white),
      ),
      confirmDismiss: (_) => _confirmDelete(context),
      onDismissed: (_) => onDismissed?.call(),
      child: tile,
    );
  }

  Future<bool> _confirmDelete(BuildContext context) async {
    final result = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Remove Server'),
        content: Text('Remove "${server.name}" from paired servers?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            style: FilledButton.styleFrom(
              backgroundColor: AppColors.statusNeedsAttention,
            ),
            child: const Text('Remove'),
          ),
        ],
      ),
    );
    return result ?? false;
  }
}

/// Small chip showing the current connection status.
class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label, required this.isConnected});

  final String label;
  final bool isConnected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    final bgColor = isConnected
        ? AppColors.statusActive.withValues(alpha: 0.12)
        : (isDark ? AppColors.shade3.withValues(alpha: 0.2) : AppColors.shade1);

    final fgColor = isConnected ? AppColors.statusActive : AppColors.shade3;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: fgColor,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

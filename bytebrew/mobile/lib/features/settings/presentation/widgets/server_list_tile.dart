import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/domain/server.dart';
import '../../../../core/infrastructure/ws/ws_connection.dart';
import '../../../../core/infrastructure/ws/ws_connection_manager.dart';
import '../../../../core/infrastructure/ws/ws_providers.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/status_indicator.dart';

/// Orange color for "connecting" / "reconnecting" state.
const _statusConnecting = Color(0xFFFFA726);

/// A list tile displaying a paired server's name and connection status.
///
/// Reads live connection state from [WsConnectionManager] to show
/// online/offline/connecting status with a colored [StatusIndicator].
class ServerListTile extends ConsumerWidget {
  const ServerListTile({super.key, required this.server, this.onDismissed});

  final Server server;

  /// Called when the user swipes to dismiss the tile.
  final VoidCallback? onDismissed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final manager = ref.watch(connectionManagerProvider);
    final connection = manager.getConnection(server.id);

    final connectionStatus = _resolveStatus(connection);
    final statusColor = _statusColor(connectionStatus);
    final subtitleText = _subtitleText(connectionStatus);

    final tile = ListTile(
      leading: StatusIndicator(color: statusColor),
      title: Text(server.name),
      subtitle: Text(subtitleText),
      trailing: _trailing(context, connectionStatus),
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

  Widget? _trailing(BuildContext context, _ServerConnectionStatus status) {
    if (status != _ServerConnectionStatus.online || server.latencyMs <= 0) {
      return null;
    }

    return Text(
      '${server.latencyMs}ms',
      style: Theme.of(context)
          .textTheme
          .bodySmall
          ?.copyWith(color: AppColors.shade3),
    );
  }

  _ServerConnectionStatus _resolveStatus(WsServerConnection? connection) {
    if (connection == null) return _ServerConnectionStatus.offline;

    return switch (connection.status) {
      WsConnectionStatus.connected => _ServerConnectionStatus.online,
      WsConnectionStatus.connecting => _ServerConnectionStatus.connecting,
      WsConnectionStatus.disconnected => _ServerConnectionStatus.offline,
      WsConnectionStatus.error => _ServerConnectionStatus.offline,
    };
  }

  Color _statusColor(_ServerConnectionStatus status) {
    return switch (status) {
      _ServerConnectionStatus.online => AppColors.statusActive,
      _ServerConnectionStatus.connecting => _statusConnecting,
      _ServerConnectionStatus.offline => AppColors.shade3,
    };
  }

  String _subtitleText(_ServerConnectionStatus status) {
    return switch (status) {
      _ServerConnectionStatus.online => 'Online',
      _ServerConnectionStatus.connecting => 'Connecting...',
      _ServerConnectionStatus.offline => 'Offline',
    };
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

/// Internal status for display purposes.
enum _ServerConnectionStatus { online, connecting, offline }

import 'package:flutter/material.dart';

import '../../../../core/infrastructure/ws/ws_connection_manager.dart';
import '../../../../core/theme/app_colors.dart';

/// Compact badge showing the connection and encryption status.
///
/// Displayed in the chat AppBar actions.
class ConnectionInfoBadge extends StatelessWidget {
  const ConnectionInfoBadge({super.key, required this.connection});

  final WsServerConnection connection;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    final isConnected = connection.status == WsConnectionStatus.connected;

    final bgColor = isDark
        ? AppColors.shade3.withValues(alpha: 0.15)
        : AppColors.shade1.withValues(alpha: 0.7);

    final fgColor = isConnected ? AppColors.statusActive : AppColors.shade3;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.cloud_outlined, size: 12, color: fgColor),
          const SizedBox(width: 3),
          Text(
            'Bridge',
            style: theme.textTheme.labelSmall?.copyWith(
              color: fgColor,
              fontWeight: FontWeight.w600,
            ),
          ),
          if (connection.server.hasEncryption) ...[
            const SizedBox(width: 4),
            Icon(Icons.lock, size: 12, color: fgColor),
          ],
        ],
      ),
    );
  }
}

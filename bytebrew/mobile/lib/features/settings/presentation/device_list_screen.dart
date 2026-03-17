import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/app_colors.dart';

/// Screen showing paired devices for a specific server.
///
/// Device management is not available over WebSocket -- this screen
/// shows a placeholder message.
class DeviceListScreen extends ConsumerWidget {
  const DeviceListScreen({super.key, required this.serverId});

  final String serverId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Paired Devices')),
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.devices, size: 48, color: AppColors.shade3),
            const SizedBox(height: 16),
            Text(
              'Device management is not available',
              style: theme.textTheme.titleMedium?.copyWith(
                color: AppColors.shade3,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              'Use the CLI to manage paired devices',
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

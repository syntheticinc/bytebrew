import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/infrastructure/ws/ws_providers.dart';
import '../../../core/theme/app_colors.dart';
import '../../sessions/application/sessions_provider.dart';
import '../../settings/application/settings_provider.dart';
import '../application/pairing_provider.dart';
import 'widgets/qr_scanner_widget.dart';

/// Screen for adding a new server by scanning the QR code shown in CLI.
class AddServerScreen extends ConsumerStatefulWidget {
  const AddServerScreen({super.key});

  @override
  ConsumerState<AddServerScreen> createState() => _AddServerScreenState();
}

class _AddServerScreenState extends ConsumerState<AddServerScreen> {
  bool _isLoading = false;
  String? _errorMessage;

  void _onQrScanned(QrPairingData data) {
    if (_isLoading) return;
    _isLoading = true;
    setState(() {});
    _doPair(data);
  }

  Future<void> _doPair(QrPairingData data) async {
    setState(() => _errorMessage = null);

    try {
      final serverPublicKey = data.serverPublicKey != null
          ? base64Decode(data.serverPublicKey!)
          : null;
      final server = await ref
          .read(pairDeviceProvider.notifier)
          .pair(
            bridgeUrl: data.bridgeUrl,
            serverId: data.serverId,
            pairingToken: data.pairingToken,
            serverPublicKey: serverPublicKey,
          );
      if (!mounted) return;

      ref.invalidate(serversProvider);
      ref.invalidate(sessionRepositoryProvider);
      context.go('/sessions');

      final manager = ref.read(connectionManagerProvider);
      // Disconnect old connection first (re-pairing replaces crypto keys).
      await manager.disconnectFromServer(server.id);
      unawaited(manager.connectToServer(server));
    } catch (e) {
      if (!mounted) return;
      setState(() => _errorMessage = _friendlyError(e));
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _showManualPairDialog() async {
    final controller = TextEditingController();
    final json = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Manual Pair'),
        content: TextField(
          key: const ValueKey('manual_pair_input'),
          controller: controller,
          decoration: const InputDecoration(
            hintText: '{"b":"...","s":"...","t":"...","k":"..."}',
            border: OutlineInputBorder(),
          ),
          maxLines: 5,
          minLines: 3,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Cancel'),
          ),
          FilledButton(
            key: const ValueKey('manual_pair_submit'),
            onPressed: () => Navigator.of(ctx).pop(controller.text),
            child: const Text('Pair'),
          ),
        ],
      ),
    );
    controller.dispose();

    if (json == null || json.trim().isEmpty) return;

    try {
      final data = QrPairingData.fromJson(json.trim());
      _onQrScanned(data);
    } on FormatException catch (e) {
      if (mounted) setState(() => _errorMessage = e.message);
    }
  }

  String _friendlyError(Object error) {
    final msg = error.toString().toLowerCase();
    if (msg.contains('timeout') || msg.contains('deadline')) {
      return 'Connection timed out. Make sure the server is running.';
    }
    if (msg.contains('connection refused') || msg.contains('unreachable')) {
      return 'Could not reach the server. Check the address and try again.';
    }
    return 'Something went wrong. Please try again.';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Scaffold(
      appBar: AppBar(title: const Text('Pair New Server')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _InstructionSection(theme: theme, isDark: isDark),
            const SizedBox(height: 24),
            if (_errorMessage != null) ...[
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.statusNeedsAttention.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: AppColors.statusNeedsAttention.withValues(alpha: 0.3),
                  ),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(
                      Icons.error_outline,
                      size: 20,
                      color: AppColors.statusNeedsAttention,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        _errorMessage!,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: AppColors.statusNeedsAttention,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
            ],
            if (_isLoading) ...[
              const SizedBox(height: 48),
              const Center(child: CircularProgressIndicator()),
              const SizedBox(height: 16),
              Center(
                child: Text(
                  'Connecting to server...',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: AppColors.shade3,
                  ),
                ),
              ),
              const SizedBox(height: 48),
            ] else ...[
              Center(
                child: QrScannerWidget(
                  key: const ValueKey('qr_scanner'),
                  onScanned: _onQrScanned,
                ),
              ),
              const SizedBox(height: 16),
              Center(
                child: Text(
                  'Point your camera at the QR code shown in CLI after typing /pair',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: AppColors.shade3,
                  ),
                  textAlign: TextAlign.center,
                ),
              ),
            ],
            if (kDebugMode) ...[
              const SizedBox(height: 16),
              Center(
                child: TextButton.icon(
                  key: const ValueKey('manual_pair_button'),
                  onPressed: _isLoading ? null : _showManualPairDialog,
                  icon: const Icon(Icons.keyboard, size: 18),
                  label: const Text('Enter code manually'),
                ),
              ),
            ],
            const SizedBox(height: 32),
            _SecurityInfo(isDark: isDark),
          ],
        ),
      ),
    );
  }
}

/// Instruction section telling the user to run the CLI command.
class _InstructionSection extends StatelessWidget {
  const _InstructionSection({required this.theme, required this.isDark});

  final ThemeData theme;
  final bool isDark;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('In ByteBrew CLI, type:', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: isDark ? AppColors.darkAlt : AppColors.shade1,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(
            '/pair',
            style: theme.textTheme.bodyLarge?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ],
    );
  }
}

/// Security information section.
class _SecurityInfo extends StatelessWidget {
  const _SecurityInfo({required this.isDark});

  final bool isDark;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.lock_outline, size: 14, color: AppColors.shade3),
            const SizedBox(width: 4),
            Text(
              'End-to-end encrypted connection',
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          'QR code contains a one-time cryptographic token',
          style: theme.textTheme.bodySmall?.copyWith(
            color: AppColors.shade3.withValues(alpha: 0.7),
          ),
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}

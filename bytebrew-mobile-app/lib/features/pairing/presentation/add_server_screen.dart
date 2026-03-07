import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/infrastructure/ws/ws_providers.dart';
import '../../../core/theme/app_colors.dart';
import '../../sessions/application/sessions_provider.dart';
import '../../settings/application/settings_provider.dart';
import '../application/pairing_provider.dart';
import 'widgets/qr_scanner_widget.dart';

/// Pairing input mode: QR scanner or manual code entry.
enum _PairingMode { qrScan, manualCode }

/// Screen for adding a new server via QR code scan or manual 6-digit code entry.
class AddServerScreen extends ConsumerStatefulWidget {
  const AddServerScreen({super.key});

  @override
  ConsumerState<AddServerScreen> createState() => _AddServerScreenState();
}

class _AddServerScreenState extends ConsumerState<AddServerScreen> {
  static const _codeLength = 6;

  final _addressController = TextEditingController();
  final _controllers = List.generate(
    _codeLength,
    (_) => TextEditingController(),
  );
  final _focusNodes = List.generate(_codeLength, (_) => FocusNode());
  final _keyListenerFocusNodes = List.generate(_codeLength, (_) => FocusNode());

  _PairingMode _mode = _PairingMode.qrScan;
  bool _isLoading = false;
  String? _errorMessage;

  /// Full token from QR scan (bypasses 6-digit code fields).
  String? _qrToken;

  /// Server ID from QR scan.
  String? _qrServerId;

  /// Server public key from QR scan (base64).
  String? _qrServerPublicKey;

  String get _code => _controllers.map((c) => c.text).join();

  bool get _isManualFormComplete =>
      _addressController.text.trim().isNotEmpty &&
      _controllers.every((c) => c.text.length == 1) &&
      _code.length == _codeLength;

  @override
  void dispose() {
    _addressController.dispose();
    for (final controller in _controllers) {
      controller.dispose();
    }
    for (final node in _focusNodes) {
      node.dispose();
    }
    for (final node in _keyListenerFocusNodes) {
      node.dispose();
    }
    super.dispose();
  }

  void _onDigitChanged(int index, String value) {
    if (value.isNotEmpty && index < _codeLength - 1) {
      _focusNodes[index + 1].requestFocus();
    }
    setState(() {});
  }

  KeyEventResult _onKeyEvent(int index, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;

    final isBackspace = event.logicalKey == LogicalKeyboardKey.backspace;
    if (!isBackspace) return KeyEventResult.ignored;
    if (index <= 0) return KeyEventResult.ignored;
    if (_controllers[index].text.isNotEmpty) return KeyEventResult.ignored;

    _controllers[index - 1].clear();
    _focusNodes[index - 1].requestFocus();
    setState(() {});
    return KeyEventResult.handled;
  }

  void _onQrScanned(QrPairingData data) {
    if (_isLoading) return; // Prevent double-scan while connecting.

    // Set loading synchronously BEFORE any async work to block subsequent
    // callbacks from the QR scanner (fires on multiple frames).
    _isLoading = true;

    // QR contains bridge URL, server ID and pairing token -- connect directly.
    _qrToken = data.pairingToken;
    _addressController.text = data.bridgeUrl;
    _qrServerId = data.serverId;
    _qrServerPublicKey = data.serverPublicKey;

    setState(() {});
    _onConnect();
  }

  Future<void> _onConnect() async {
    final pairingCode = _qrToken ?? _code;
    final address = _addressController.text.trim();

    // For manual mode, require complete form. For QR, we have full token.
    if (_qrToken == null && !_isManualFormComplete) return;
    if (address.isEmpty || pairingCode.isEmpty) return;

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      debugPrint('[AddServer] Starting pair...');
      final serverPublicKey = _qrServerPublicKey != null
          ? base64Decode(_qrServerPublicKey!)
          : null;
      final server = await ref
          .read(pairDeviceProvider.notifier)
          .pair(
            bridgeUrl: address,
            serverId: _qrServerId ?? '',
            pairingToken: pairingCode,
            serverPublicKey: serverPublicKey,
          );
      debugPrint('[AddServer] Pair success: ${server.id}');
      if (!mounted) return;

      // Invalidate cached providers so sessions screen picks up new connection.
      ref.invalidate(serversProvider);
      ref.invalidate(sessionRepositoryProvider);

      // Navigate immediately.
      context.go('/sessions');

      // Connect via ConnectionManager in background (fire-and-forget).
      debugPrint('[AddServer] Starting background WS connect...');
      final manager = ref.read(connectionManagerProvider);
      unawaited(manager.connectToServer(server));
    } catch (e, st) {
      debugPrint('[AddServer] ERROR: $e');
      debugPrint('[AddServer] Stack: $st');
      if (!mounted) return;
      setState(() => _errorMessage = _friendlyError(e));
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
          _qrToken = null;
          _qrServerId = null;
          _qrServerPublicKey = null;
        });
      }
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
            // Mode selector
            Center(
              child: SegmentedButton<_PairingMode>(
                segments: const [
                  ButtonSegment(
                    value: _PairingMode.qrScan,
                    label: Text('QR Scan'),
                    icon: Icon(Icons.qr_code_scanner),
                  ),
                  ButtonSegment(
                    value: _PairingMode.manualCode,
                    label: Text('Manual Code'),
                    icon: Icon(Icons.keyboard),
                  ),
                ],
                selected: {_mode},
                onSelectionChanged: (selection) {
                  setState(() {
                    _mode = selection.first;
                    _errorMessage = null;
                  });
                },
                style: ButtonStyle(
                  backgroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.selected)) {
                      return AppColors.accent;
                    }
                    return null;
                  }),
                  foregroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.selected)) {
                      return AppColors.light;
                    }
                    return null;
                  }),
                ),
              ),
            ),
            const SizedBox(height: 24),
            // Error message
            if (_errorMessage != null) ...[
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.statusNeedsAttention.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: AppColors.statusNeedsAttention.withValues(
                      alpha: 0.3,
                    ),
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
            // QR scan mode
            if (_mode == _PairingMode.qrScan) ...[
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
                Center(child: QrScannerWidget(onScanned: _onQrScanned)),
                const SizedBox(height: 16),
                Center(
                  child: Text(
                    'Point your camera at the QR code shown by bytebrew mobile-pair',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: AppColors.shade3,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ),
              ],
            ],
            // Manual code mode
            if (_mode == _PairingMode.manualCode) ...[
              _ServerAddressField(
                controller: _addressController,
                isDark: isDark,
                onChanged: (_) => setState(() {}),
              ),
              const SizedBox(height: 24),
              Center(
                child: Text(
                  'Enter the 6-digit pairing code',
                  style: theme.textTheme.titleSmall,
                ),
              ),
              const SizedBox(height: 12),
              _CodeInput(
                controllers: _controllers,
                focusNodes: _focusNodes,
                keyListenerFocusNodes: _keyListenerFocusNodes,
                onChanged: _onDigitChanged,
                onKeyEvent: _onKeyEvent,
                isDark: isDark,
              ),
              const SizedBox(height: 32),
              SizedBox(
                width: double.infinity,
                height: 48,
                child: FilledButton(
                  onPressed: _isManualFormComplete && !_isLoading
                      ? _onConnect
                      : null,
                  child: _isLoading
                      ? const SizedBox(
                          width: 24,
                          height: 24,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: AppColors.light,
                          ),
                        )
                      : const Text('Connect'),
                ),
              ),
            ],
            const SizedBox(height: 32),
            const _SecurityInfo(),
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
        Text('Run in your terminal:', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: isDark ? AppColors.darkAlt : AppColors.shade1,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(
            'bytebrew mobile-pair',
            style: theme.textTheme.bodyLarge?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ],
    );
  }
}

/// Text field for entering the server's LAN address (e.g. "192.168.1.5").
class _ServerAddressField extends StatelessWidget {
  const _ServerAddressField({
    required this.controller,
    required this.isDark,
    required this.onChanged,
  });

  final TextEditingController controller;
  final bool isDark;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Server Address', style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        TextField(
          controller: controller,
          keyboardType: TextInputType.url,
          decoration: InputDecoration(
            hintText: 'e.g. 192.168.1.5',
            filled: true,
            fillColor: isDark ? AppColors.darkAlt : AppColors.white,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(
                color: AppColors.shade3.withValues(alpha: 0.3),
              ),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(
                color: AppColors.shade3.withValues(alpha: 0.3),
              ),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.accent, width: 1.5),
            ),
          ),
          onChanged: onChanged,
        ),
      ],
    );
  }
}

/// Row of 6 individual digit input fields for the pairing code.
class _CodeInput extends StatelessWidget {
  const _CodeInput({
    required this.controllers,
    required this.focusNodes,
    required this.keyListenerFocusNodes,
    required this.onChanged,
    required this.onKeyEvent,
    required this.isDark,
  });

  final List<TextEditingController> controllers;
  final List<FocusNode> focusNodes;
  final List<FocusNode> keyListenerFocusNodes;
  final void Function(int index, String value) onChanged;
  final KeyEventResult Function(int index, KeyEvent event) onKeyEvent;
  final bool isDark;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: List.generate(controllers.length, (index) {
        return Padding(
          padding: EdgeInsets.only(left: index == 0 ? 0 : 8),
          child: SizedBox(
            width: 48,
            height: 56,
            child: KeyboardListener(
              focusNode: keyListenerFocusNodes[index],
              onKeyEvent: (event) => onKeyEvent(index, event),
              child: TextField(
                controller: controllers[index],
                focusNode: focusNodes[index],
                textAlign: TextAlign.center,
                keyboardType: TextInputType.number,
                maxLength: 1,
                style: theme.textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                decoration: InputDecoration(
                  counterText: '',
                  filled: true,
                  fillColor: isDark ? AppColors.darkAlt : AppColors.white,
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(
                      color: AppColors.shade3.withValues(alpha: 0.3),
                    ),
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(
                      color: AppColors.shade3.withValues(alpha: 0.3),
                    ),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(
                      color: AppColors.accent,
                      width: 1.5,
                    ),
                  ),
                ),
                onChanged: (value) => onChanged(index, value),
              ),
            ),
          ),
        );
      }),
    );
  }
}

/// Security information section.
class _SecurityInfo extends StatelessWidget {
  const _SecurityInfo();

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
              'Encrypted connection via bridge relay',
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          'End-to-end encrypted connection through a secure relay',
          style: theme.textTheme.bodySmall?.copyWith(
            color: AppColors.shade3.withValues(alpha: 0.7),
          ),
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}

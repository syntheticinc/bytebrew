import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_colors.dart';

/// Authentication screen for connecting via QR code scan or manual code entry.
class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key});

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> {
  final _codeController = TextEditingController();
  bool _isConnecting = false;

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  void _connect() {
    final code = _codeController.text.trim();
    if (code.length < 6 || _isConnecting) return;

    setState(() => _isConnecting = true);
    Future.delayed(const Duration(seconds: 1), () {
      if (!mounted) return;
      context.go('/sessions');
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Scaffold(
      backgroundColor: isDark ? AppColors.dark : AppColors.light,
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                _Wordmark(theme: theme, isDark: isDark),
                const SizedBox(height: 40),
                _QrViewfinder(isDark: isDark),
                const SizedBox(height: 24),
                _ScanInstruction(theme: theme),
                const SizedBox(height: 28),
                _OrDivider(theme: theme),
                const SizedBox(height: 28),
                _CodeField(
                  controller: _codeController,
                  isDark: isDark,
                  theme: theme,
                  onChanged: (_) => setState(() {}),
                  onSubmitted: (_) => _connect(),
                ),
                const SizedBox(height: 24),
                _ConnectButton(
                  isEnabled: _codeController.text.trim().length >= 6,
                  isConnecting: _isConnecting,
                  onPressed: _connect,
                ),
                const SizedBox(height: 32),
                const _SecurityNote(),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Brand wordmark with subtitle.
class _Wordmark extends StatelessWidget {
  const _Wordmark({required this.theme, required this.isDark});

  final ThemeData theme;
  final bool isDark;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(
          'Byte Brew',
          style: theme.textTheme.headlineLarge?.copyWith(
            fontWeight: FontWeight.bold,
            color: isDark ? AppColors.light : AppColors.dark,
            letterSpacing: -0.5,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          'Your AI agents, everywhere',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: AppColors.shade3,
            letterSpacing: 0.5,
          ),
        ),
      ],
    );
  }
}

/// QR scanner placeholder with dashed border.
class _QrViewfinder extends StatelessWidget {
  const _QrViewfinder({required this.isDark});

  final bool isDark;

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      painter: _DashedBorderPainter(
        color: AppColors.shade3.withValues(alpha: 0.5),
        borderRadius: 16,
        dashWidth: 8,
        dashGap: 6,
        strokeWidth: 1.5,
      ),
      child: Container(
        width: 250,
        height: 250,
        decoration: BoxDecoration(
          color: isDark ? AppColors.darkAlt : AppColors.white,
          borderRadius: BorderRadius.circular(16),
        ),
        child: const Center(
          child: Icon(Icons.qr_code_scanner, size: 48, color: AppColors.shade3),
        ),
      ),
    );
  }
}

/// CustomPainter that draws a dashed rounded rectangle border.
class _DashedBorderPainter extends CustomPainter {
  const _DashedBorderPainter({
    required this.color,
    required this.borderRadius,
    required this.dashWidth,
    required this.dashGap,
    required this.strokeWidth,
  });

  final Color color;
  final double borderRadius;
  final double dashWidth;
  final double dashGap;
  final double strokeWidth;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = strokeWidth
      ..style = PaintingStyle.stroke;

    final rrect = RRect.fromRectAndRadius(
      Rect.fromLTWH(0, 0, size.width, size.height),
      Radius.circular(borderRadius),
    );

    final path = Path()..addRRect(rrect);
    final metrics = path.computeMetrics();

    for (final metric in metrics) {
      var distance = 0.0;
      while (distance < metric.length) {
        final end = (distance + dashWidth).clamp(0.0, metric.length);
        final segment = metric.extractPath(distance, end);
        canvas.drawPath(segment, paint);
        distance += dashWidth + dashGap;
      }
    }
  }

  @override
  bool shouldRepaint(covariant _DashedBorderPainter oldDelegate) {
    return color != oldDelegate.color ||
        borderRadius != oldDelegate.borderRadius ||
        dashWidth != oldDelegate.dashWidth ||
        dashGap != oldDelegate.dashGap ||
        strokeWidth != oldDelegate.strokeWidth;
  }
}

/// Instruction text below the QR viewfinder.
class _ScanInstruction extends StatelessWidget {
  const _ScanInstruction({required this.theme});

  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Text(
      'Scan the QR code from\nyour ByteBrew CLI to\nconnect this device',
      textAlign: TextAlign.center,
      style: theme.textTheme.bodyMedium?.copyWith(
        color: AppColors.shade3,
        height: 1.5,
      ),
    );
  }
}

/// Horizontal divider with centered "or enter code" text.
class _OrDivider extends StatelessWidget {
  const _OrDivider({required this.theme});

  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        const Expanded(child: Divider()),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            'or enter code',
            style: theme.textTheme.bodySmall?.copyWith(color: AppColors.shade3),
          ),
        ),
        const Expanded(child: Divider()),
      ],
    );
  }
}

/// Single text field for entering a 6-digit connection code.
class _CodeField extends StatelessWidget {
  const _CodeField({
    required this.controller,
    required this.isDark,
    required this.theme,
    required this.onChanged,
    required this.onSubmitted,
  });

  final TextEditingController controller;
  final bool isDark;
  final ThemeData theme;
  final ValueChanged<String> onChanged;
  final ValueChanged<String> onSubmitted;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      textAlign: TextAlign.center,
      maxLength: 6,
      keyboardType: TextInputType.number,
      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
      style: theme.textTheme.headlineSmall?.copyWith(
        fontWeight: FontWeight.bold,
        letterSpacing: 8,
      ),
      decoration: InputDecoration(
        counterText: '',
        hintText: '------',
        hintStyle: theme.textTheme.headlineSmall?.copyWith(
          color: AppColors.shade3.withValues(alpha: 0.4),
          letterSpacing: 8,
          fontWeight: FontWeight.bold,
        ),
        filled: true,
        fillColor: isDark ? AppColors.darkAlt : AppColors.white,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(
            color: AppColors.shade3.withValues(alpha: 0.3),
          ),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(
            color: AppColors.shade3.withValues(alpha: 0.3),
          ),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.accent, width: 1.5),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 16,
        ),
      ),
      onChanged: onChanged,
      onSubmitted: onSubmitted,
    );
  }
}

/// Full-width connect button with loading state.
class _ConnectButton extends StatelessWidget {
  const _ConnectButton({
    required this.isEnabled,
    required this.isConnecting,
    required this.onPressed,
  });

  final bool isEnabled;
  final bool isConnecting;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      height: 48,
      child: FilledButton(
        onPressed: isEnabled && !isConnecting ? onPressed : null,
        child: isConnecting
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
    );
  }
}

/// Security note at the bottom with lock icon.
class _SecurityNote extends StatelessWidget {
  const _SecurityNote();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Icon(Icons.lock_outline, size: 14, color: AppColors.shade3),
        const SizedBox(width: 6),
        Text(
          'End-to-end encrypted',
          style: theme.textTheme.bodySmall?.copyWith(color: AppColors.shade3),
        ),
      ],
    );
  }
}

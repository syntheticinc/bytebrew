import 'package:flutter/material.dart';

import '../domain/session.dart';
import '../theme/app_colors.dart';

/// An expressive status indicator dot with animations based on [SessionStatus].
///
/// - [SessionStatus.needsAttention]: pulsing red dot with animated opacity.
/// - [SessionStatus.active]: green dot with a subtle outer ring/glow.
/// - [SessionStatus.idle]: static, smaller grey dot.
class AnimatedStatusIndicator extends StatefulWidget {
  const AnimatedStatusIndicator({
    super.key,
    required this.status,
    this.size = 10,
  });

  final SessionStatus status;
  final double size;

  @override
  State<AnimatedStatusIndicator> createState() =>
      _AnimatedStatusIndicatorState();
}

class _AnimatedStatusIndicatorState extends State<AnimatedStatusIndicator>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _pulseAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
    _pulseAnimation = Tween<double>(
      begin: 0.5,
      end: 1.0,
    ).animate(CurvedAnimation(parent: _controller, curve: Curves.easeInOut));

    if (widget.status == SessionStatus.needsAttention) {
      _controller.repeat(reverse: true);
    }
  }

  @override
  void didUpdateWidget(covariant AnimatedStatusIndicator oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.status != widget.status) {
      if (widget.status == SessionStatus.needsAttention) {
        _controller.repeat(reverse: true);
      } else {
        _controller.stop();
        _controller.value = 1.0;
      }
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final color = AppColors.statusColor(widget.status);

    return switch (widget.status) {
      SessionStatus.needsAttention => _buildPulsingDot(color),
      SessionStatus.active => _buildGlowDot(color),
      SessionStatus.idle => _buildStaticDot(color),
    };
  }

  Widget _buildPulsingDot(Color color) {
    return AnimatedBuilder(
      animation: _pulseAnimation,
      builder: (context, child) {
        return Opacity(
          opacity: _pulseAnimation.value,
          child: Container(
            width: widget.size,
            height: widget.size,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
        );
      },
    );
  }

  Widget _buildGlowDot(Color color) {
    return SizedBox(
      width: widget.size + 4,
      height: widget.size + 4,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Container(
            width: widget.size + 4,
            height: widget.size + 4,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.2),
              shape: BoxShape.circle,
            ),
          ),
          Container(
            width: widget.size,
            height: widget.size,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
        ],
      ),
    );
  }

  Widget _buildStaticDot(Color color) {
    final smallSize = widget.size * 0.8;
    return SizedBox(
      width: widget.size,
      height: widget.size,
      child: Center(
        child: Container(
          width: smallSize,
          height: smallSize,
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.6),
            shape: BoxShape.circle,
          ),
        ),
      ),
    );
  }
}

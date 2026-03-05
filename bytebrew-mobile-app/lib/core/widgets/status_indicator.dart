import 'package:flutter/material.dart';

/// A small colored circle indicating status.
///
/// Used in session cards and server lists to visually convey status
/// (active, idle, needs attention, etc.).
class StatusIndicator extends StatelessWidget {
  const StatusIndicator({super.key, required this.color, this.size = 10});

  final Color color;
  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

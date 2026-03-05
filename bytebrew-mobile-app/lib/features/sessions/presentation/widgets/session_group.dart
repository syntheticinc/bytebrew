import 'package:flutter/material.dart';

import '../../../../core/domain/session.dart';
import '../../../../core/theme/app_colors.dart';
import 'session_card.dart';

/// A group of sessions with uppercase monospace header.
class SessionGroup extends StatelessWidget {
  const SessionGroup({super.key, required this.status, required this.sessions});

  final SessionStatus status;
  final List<Session> sessions;

  static String _statusLabel(SessionStatus status) {
    return switch (status) {
      SessionStatus.needsAttention => 'ACTION REQUIRED',
      SessionStatus.active => 'IN PROGRESS',
      SessionStatus.idle => 'RECENT',
    };
  }

  static Color _headerColor(SessionStatus status) {
    return switch (status) {
      SessionStatus.needsAttention => AppColors.accent,
      SessionStatus.active => AppColors.statusActive,
      SessionStatus.idle => AppColors.shade3,
    };
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = _headerColor(status);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
          child: Text(
            '${_statusLabel(status)} (${sessions.length})',
            style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w600,
              letterSpacing: 2,
              color: color,
            ),
          ),
        ),
        ...sessions.map((session) => SessionCard(session: session)),
      ],
    );
  }
}

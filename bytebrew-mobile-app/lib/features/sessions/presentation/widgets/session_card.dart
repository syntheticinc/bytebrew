import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/domain/session.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/utils/time_ago.dart';
import '../../../../core/widgets/animated_status_indicator.dart';

/// A compact branded session card with left border status indicator.
///
/// Two-line layout:
/// ```
/// [*] ProjectName   taskName...   2m ago
///     server-1                 [Waiting]
/// ```
///
/// - needsAttention: left border 2px accent
/// - active: left border 2px green
/// - idle: no left border, slightly muted text
class SessionCard extends StatelessWidget {
  const SessionCard({super.key, required this.session});

  final Session session;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
      child: Container(
        decoration: BoxDecoration(
          color: isDark ? AppColors.darkAlt : AppColors.white,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: AppColors.shade3.withValues(alpha: 0.15)),
        ),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(8),
          child: Container(
            decoration: BoxDecoration(border: Border(left: _leftBorder())),
            child: InkWell(
              borderRadius: BorderRadius.circular(8),
              onTap: () => context.push('/chat/${session.id}'),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 8,
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _PrimaryRow(session: session, theme: theme),
                    const SizedBox(height: 2),
                    _SecondaryRow(session: session, theme: theme),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  BorderSide _leftBorder() {
    return switch (session.status) {
      SessionStatus.needsAttention => const BorderSide(
        color: AppColors.accent,
        width: 2,
      ),
      SessionStatus.active => const BorderSide(
        color: AppColors.statusActive,
        width: 2,
      ),
      SessionStatus.idle => BorderSide.none,
    };
  }
}

/// Primary row: status dot + projectName + currentTask (truncated) + timeAgo.
class _PrimaryRow extends StatelessWidget {
  const _PrimaryRow({required this.session, required this.theme});

  final Session session;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        AnimatedStatusIndicator(status: session.status, size: 8),
        const SizedBox(width: 8),
        Text(
          session.projectName,
          style: theme.textTheme.bodySmall?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        if (session.currentTask != null) ...[
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              session.currentTask!,
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ] else
          const Spacer(),
        const SizedBox(width: 8),
        Text(
          timeAgo(session.lastActivityAt),
          style: theme.textTheme.bodySmall?.copyWith(
            color: AppColors.shade3,
            fontSize: 11,
          ),
        ),
      ],
    );
  }
}

/// Secondary row: serverName + optional "Waiting" badge.
class _SecondaryRow extends StatelessWidget {
  const _SecondaryRow({required this.session, required this.theme});

  final Session session;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 16),
      child: Row(
        children: [
          Expanded(
            child: Text(
              session.serverName,
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
                fontSize: 11,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (session.hasAskUser) const _WaitingBadge(),
        ],
      ),
    );
  }
}

/// Compact accent badge indicating the agent is waiting for user input.
class _WaitingBadge extends StatelessWidget {
  const _WaitingBadge();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: AppColors.accent.withValues(alpha: 0.3)),
      ),
      child: Text(
        'Waiting',
        style: theme.textTheme.labelSmall?.copyWith(
          color: AppColors.accent,
          fontWeight: FontWeight.w600,
          fontSize: 10,
        ),
      ),
    );
  }
}

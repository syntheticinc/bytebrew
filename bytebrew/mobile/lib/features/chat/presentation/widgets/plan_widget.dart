import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/domain/chat_message.dart';
import '../../../../core/domain/plan.dart';
import '../../../../core/theme/app_colors.dart';

/// Inline plan card with brand styling.
class PlanWidget extends StatelessWidget {
  const PlanWidget({super.key, required this.message, required this.sessionId});

  final ChatMessage message;
  final String sessionId;

  static const _maxVisibleSteps = 4;

  @override
  Widget build(BuildContext context) {
    final plan = message.plan;
    if (plan == null) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Container(
        decoration: BoxDecoration(
          color: isDark ? AppColors.darkAlt : AppColors.white,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: AppColors.shade3.withValues(alpha: 0.15)),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () => context.push('/plan/$sessionId'),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    const Icon(
                      Icons.checklist,
                      size: 18,
                      color: AppColors.accent,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        plan.goal,
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                    Icon(
                      Icons.chevron_right,
                      size: 20,
                      color: AppColors.shade3,
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                ClipRRect(
                  borderRadius: BorderRadius.circular(3),
                  child: LinearProgressIndicator(
                    value: plan.progress,
                    minHeight: 6,
                    color: AppColors.accent,
                    backgroundColor: AppColors.accent.withValues(alpha: 0.12),
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '${(plan.progress * 100).round()}% complete',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: AppColors.shade3,
                  ),
                ),
                const SizedBox(height: 8),
                ..._buildVisibleSteps(theme),
                if (plan.steps.length > _maxVisibleSteps)
                  Padding(
                    padding: const EdgeInsets.only(top: 4),
                    child: Text(
                      '${plan.steps.length - _maxVisibleSteps} more steps...',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: AppColors.shade3,
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  List<Widget> _buildVisibleSteps(ThemeData theme) {
    final plan = message.plan!;
    final visibleSteps = plan.steps.take(_maxVisibleSteps);

    return visibleSteps.map((step) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Row(
          children: [
            _buildStepIcon(step.status),
            const SizedBox(width: 8),
            Expanded(
              child: Text(step.description, style: theme.textTheme.bodySmall),
            ),
          ],
        ),
      );
    }).toList();
  }

  Widget _buildStepIcon(PlanStepStatus status) {
    return switch (status) {
      PlanStepStatus.completed => const Icon(
        Icons.check_circle,
        size: 16,
        color: AppColors.statusActive,
      ),
      PlanStepStatus.inProgress => const SizedBox(
        width: 16,
        height: 16,
        child: CircularProgressIndicator(
          strokeWidth: 2,
          color: AppColors.accent,
        ),
      ),
      PlanStepStatus.pending => Container(
        width: 8,
        height: 8,
        margin: const EdgeInsets.all(4),
        decoration: BoxDecoration(
          color: AppColors.shade3.withValues(alpha: 0.4),
          shape: BoxShape.circle,
        ),
      ),
      PlanStepStatus.failed => const Icon(
        Icons.cancel,
        size: 16,
        color: AppColors.statusNeedsAttention,
      ),
    };
  }
}

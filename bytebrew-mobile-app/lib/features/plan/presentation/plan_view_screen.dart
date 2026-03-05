import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';

/// Full plan view screen with branded styling.
class PlanViewScreen extends ConsumerWidget {
  const PlanViewScreen({super.key, required this.sessionId});

  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final plan = ref.watch(activePlanProvider(sessionId));
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Plan'),
            if (plan != null)
              Text(
                plan.goal,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: AppColors.shade3,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
          ],
        ),
      ),
      body: plan == null ? const _EmptyPlanBody() : _PlanContent(plan: plan),
    );
  }
}

class _EmptyPlanBody extends StatelessWidget {
  const _EmptyPlanBody();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.checklist_outlined,
            size: 48,
            color: AppColors.shade3.withValues(alpha: 0.5),
          ),
          const SizedBox(height: 16),
          Text(
            'No active plan',
            style: theme.textTheme.titleMedium?.copyWith(
              color: AppColors.shade3,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'The agent will create a plan when needed',
            style: theme.textTheme.bodySmall?.copyWith(
              color: AppColors.shade3.withValues(alpha: 0.7),
            ),
          ),
        ],
      ),
    );
  }
}

class _PlanContent extends StatelessWidget {
  const _PlanContent({required this.plan});

  final PlanData plan;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final completedCount = plan.steps
        .where((s) => s.status == PlanStepStatus.completed)
        .length;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                plan.goal,
                style: theme.textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 16),
              ClipRRect(
                borderRadius: BorderRadius.circular(3),
                child: LinearProgressIndicator(
                  value: plan.progress,
                  minHeight: 6,
                  color: AppColors.accent,
                  backgroundColor: AppColors.accent.withValues(alpha: 0.12),
                ),
              ),
              const SizedBox(height: 8),
              Text(
                '${(plan.progress * 100).round()}% complete  \u00b7  $completedCount/${plan.steps.length} steps',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: AppColors.shade3,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Expanded(
          child: ListView.separated(
            itemCount: plan.steps.length,
            separatorBuilder: (_, _) => Divider(
              indent: 56,
              height: 1,
              color: AppColors.shade3.withValues(alpha: 0.15),
            ),
            itemBuilder: (context, index) {
              final step = plan.steps[index];
              return _PlanStepTile(step: step);
            },
          ),
        ),
      ],
    );
  }
}

class _PlanStepTile extends StatelessWidget {
  const _PlanStepTile({required this.step});

  final PlanStep step;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return ListTile(
      leading: _buildLeadingIcon(),
      title: Text(step.description, style: _titleStyle(theme)),
      subtitle: _buildSubtitle(theme),
    );
  }

  Widget _buildLeadingIcon() {
    return switch (step.status) {
      PlanStepStatus.completed => const Icon(
        Icons.check_circle,
        color: AppColors.statusActive,
        size: 24,
      ),
      PlanStepStatus.inProgress => const SizedBox(
        width: 24,
        height: 24,
        child: CircularProgressIndicator(
          strokeWidth: 2.5,
          color: AppColors.accent,
        ),
      ),
      PlanStepStatus.pending => Icon(
        Icons.circle_outlined,
        color: AppColors.shade3,
        size: 24,
      ),
    };
  }

  TextStyle? _titleStyle(ThemeData theme) {
    final base = theme.textTheme.bodyLarge;
    return switch (step.status) {
      PlanStepStatus.completed => base?.copyWith(
        decoration: TextDecoration.lineThrough,
        color: AppColors.shade3,
      ),
      PlanStepStatus.inProgress => base?.copyWith(fontWeight: FontWeight.bold),
      PlanStepStatus.pending => base,
    };
  }

  Widget? _buildSubtitle(ThemeData theme) {
    if (step.status == PlanStepStatus.completed && step.completedAt != null) {
      final formatted = DateFormat('MMM d, HH:mm').format(step.completedAt!);
      return Text(
        formatted,
        style: theme.textTheme.bodySmall?.copyWith(color: AppColors.shade3),
      );
    }

    if (step.status == PlanStepStatus.inProgress) {
      return Text(
        'In progress...',
        style: theme.textTheme.bodySmall?.copyWith(color: AppColors.accent),
      );
    }

    return null;
  }
}

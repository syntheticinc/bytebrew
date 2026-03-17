import 'package:flutter/material.dart';

import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';

/// Modal bottom sheet displaying full tool call details with branded styling.
class ToolDetailSheet extends StatelessWidget {
  const ToolDetailSheet._({
    required this.toolCall,
    required this.scrollController,
  });

  /// Test-only constructor for widget tests.
  @visibleForTesting
  const ToolDetailSheet.testOnly({
    super.key,
    required this.toolCall,
    required this.scrollController,
  });

  final ToolCallData toolCall;
  final ScrollController scrollController;

  static void show(BuildContext context, ToolCallData toolCall) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) => DraggableScrollableSheet(
        initialChildSize: 0.6,
        minChildSize: 0.3,
        maxChildSize: 0.9,
        expand: false,
        builder: (_, scrollController) => ToolDetailSheet._(
          toolCall: toolCall,
          scrollController: scrollController,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return ListView(
      controller: scrollController,
      padding: EdgeInsets.zero,
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header: tool name + status chip
              Row(
                children: [
                  Expanded(
                    child: Text(
                      toolCall.toolName,
                      style: theme.textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                  _buildStatusChip(theme),
                ],
              ),
              const SizedBox(height: 16),

              // Arguments section
              Text(
                'ARGUMENTS',
                style: theme.textTheme.labelSmall?.copyWith(
                  fontWeight: FontWeight.w600,
                  letterSpacing: 2,
                  color: AppColors.shade3,
                ),
              ),
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: isDark ? AppColors.darkAlt : AppColors.shade1,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final entry in toolCall.arguments.entries)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 4),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              '${entry.key}: ',
                              style: theme.textTheme.bodySmall?.copyWith(
                                fontWeight: FontWeight.w600,
                                color: AppColors.shade3,
                              ),
                            ),
                            Expanded(
                              child: Text(
                                entry.value,
                                style: theme.textTheme.bodySmall,
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),

              // Result section
              if (toolCall.fullResult != null || toolCall.result != null) ...[
                const SizedBox(height: 16),
                Text(
                  'RESULT',
                  style: theme.textTheme.labelSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                    letterSpacing: 2,
                    color: AppColors.shade3,
                  ),
                ),
                const SizedBox(height: 8),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: isDark ? AppColors.darkAlt : AppColors.shade1,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    toolCall.fullResult ?? toolCall.result ?? '',
                    style: theme.textTheme.bodySmall,
                  ),
                ),
              ],

              // Error section
              if (toolCall.error != null) ...[
                const SizedBox(height: 16),
                Text(
                  'ERROR',
                  style: theme.textTheme.labelSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                    letterSpacing: 2,
                    color: theme.colorScheme.error,
                  ),
                ),
                const SizedBox(height: 8),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.errorContainer,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    toolCall.error!,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onErrorContainer,
                    ),
                  ),
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildStatusChip(ThemeData theme) {
    final (String label, Color color) = switch (toolCall.status) {
      ToolCallStatus.running => ('Running', AppColors.shade3),
      ToolCallStatus.completed => ('Completed', AppColors.statusActive),
      ToolCallStatus.failed => ('Failed', AppColors.accent),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: color,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

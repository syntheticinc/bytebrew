import 'package:flutter/material.dart';

import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/tool_detail/presentation/tool_detail_sheet.dart';

/// CLI-style tool call display:
/// ● tool_name(args)
///   └ result summary
class ToolCallWidget extends StatelessWidget {
  const ToolCallWidget({super.key, required this.message});

  final ChatMessage message;

  @override
  Widget build(BuildContext context) {
    final toolCall = message.toolCall;
    if (toolCall == null) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: () => ToolDetailSheet.show(context, toolCall),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: isDark ? AppColors.darkAlt : AppColors.shade1,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              _buildToolLine(toolCall, theme),
              if (toolCall.result != null) _buildResultLine(toolCall, theme),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildToolLine(ToolCallData toolCall, ThemeData theme) {
    final argsText = toolCall.arguments.values.isNotEmpty
        ? '(${toolCall.arguments.values.first})'
        : '';

    return Row(
      children: [
        _buildBullet(toolCall.status),
        const SizedBox(width: 8),
        Text(
          toolCall.toolName,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w500,
          ),
        ),
        if (argsText.isNotEmpty)
          Flexible(
            child: Text(
              argsText,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: AppColors.shade3,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        if (toolCall.status == ToolCallStatus.running)
          Padding(
            padding: const EdgeInsets.only(left: 8),
            child: SizedBox(
              width: 12,
              height: 12,
              child: CircularProgressIndicator(
                strokeWidth: 1.5,
                color: AppColors.shade3,
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildResultLine(ToolCallData toolCall, ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.only(left: 6, top: 2),
      child: Row(
        children: [
          Text(
            '  \u2514 ',
            style: theme.textTheme.bodySmall?.copyWith(color: AppColors.shade3),
          ),
          Expanded(
            child: Text(
              toolCall.result!,
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBullet(ToolCallStatus status) {
    final color = switch (status) {
      ToolCallStatus.running => AppColors.shade3,
      ToolCallStatus.completed => AppColors.statusActive,
      ToolCallStatus.failed => AppColors.accent,
    };

    return Text('\u25CF', style: TextStyle(color: color, fontSize: 12));
  }
}

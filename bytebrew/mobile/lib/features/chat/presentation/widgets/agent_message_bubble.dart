import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

import '../../../../core/domain/chat_message.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/code_highlighter.dart';

/// A flat agent message with accent indicator and branded markdown rendering.
class AgentMessageBubble extends StatelessWidget {
  const AgentMessageBubble({super.key, required this.message});

  final ChatMessage message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final isDark = theme.brightness == Brightness.dark;

    return Padding(
      padding: const EdgeInsets.only(left: 12, right: 16, top: 4, bottom: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Small accent line indicator instead of avatar
          Container(
            width: 3,
            height: 16,
            margin: const EdgeInsets.only(top: 4),
            decoration: BoxDecoration(
              color: AppColors.accent,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: MarkdownBody(
              data: message.content,
              shrinkWrap: true,
              syntaxHighlighter: CodeHighlighter(isDark: isDark),
              styleSheet: MarkdownStyleSheet(
                p: theme.textTheme.bodyMedium?.copyWith(
                  color: colorScheme.onSurface,
                  height: 1.5,
                ),
                h1: theme.textTheme.headlineSmall?.copyWith(
                  color: colorScheme.onSurface,
                  fontWeight: FontWeight.w700,
                ),
                h2: theme.textTheme.titleLarge?.copyWith(
                  color: colorScheme.onSurface,
                  fontWeight: FontWeight.w600,
                ),
                h3: theme.textTheme.titleMedium?.copyWith(
                  color: colorScheme.onSurface,
                  fontWeight: FontWeight.w600,
                ),
                strong: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                  color: colorScheme.onSurface,
                ),
                em: theme.textTheme.bodyMedium?.copyWith(
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurface,
                ),
                listBullet: theme.textTheme.bodyMedium?.copyWith(
                  color: colorScheme.onSurface,
                ),
                // Code block styling
                code: theme.textTheme.bodySmall?.copyWith(
                  color: colorScheme.onSurface,
                  backgroundColor: isDark
                      ? AppColors.darkAlt
                      : AppColors.shade1,
                ),
                codeblockDecoration: BoxDecoration(
                  color: isDark ? AppColors.darkAlt : AppColors.shade1,
                  borderRadius: BorderRadius.circular(8),
                ),
                codeblockPadding: const EdgeInsets.all(12),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

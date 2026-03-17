import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/domain/ask_user.dart';
import '../../../../core/domain/chat_message.dart';
import '../../../../core/theme/app_colors.dart';
import '../../application/chat_provider.dart';

/// Question from the agent with accent-bordered card and full-width options.
class AskUserWidget extends ConsumerWidget {
  const AskUserWidget({
    super.key,
    required this.message,
    required this.sessionId,
  });

  final ChatMessage message;
  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final askUser = message.askUser;
    if (askUser == null) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);

    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.05),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.accent),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            askUser.question,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 12),
          if (askUser.status == AskUserStatus.pending)
            _PendingOptions(
              askUser: askUser,
              onAnswer: (text) {
                if (text.isEmpty) return;
                ref
                    .read(chatMessagesProvider(sessionId).notifier)
                    .answerAskUser(askUser.id, text);
              },
              theme: theme,
            ),
          if (askUser.status == AskUserStatus.answered)
            _AnsweredBadge(askUser: askUser, theme: theme),
        ],
      ),
    );
  }
}

class _PendingOptions extends StatelessWidget {
  const _PendingOptions({
    required this.askUser,
    required this.onAnswer,
    required this.theme,
  });

  final AskUserData askUser;
  final ValueChanged<String> onAnswer;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: 6,
      children: [
        for (final option in askUser.options)
          _OptionTile(
            text: option,
            onTap: () => onAnswer(option),
            theme: theme,
          ),
      ],
    );
  }
}

class _OptionTile extends StatelessWidget {
  const _OptionTile({
    required this.text,
    required this.onTap,
    required this.theme,
  });

  final String text;
  final VoidCallback onTap;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    final isDark = theme.brightness == Brightness.dark;

    return Material(
      color: isDark ? AppColors.darkAlt : AppColors.white,
      borderRadius: BorderRadius.circular(8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: AppColors.accent.withValues(alpha: 0.3)),
          ),
          child: Text(
            text,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
        ),
      ),
    );
  }
}

class _AnsweredBadge extends StatelessWidget {
  const _AnsweredBadge({required this.askUser, required this.theme});

  final AskUserData askUser;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: AppColors.statusActive.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          const Icon(
            Icons.check_circle,
            size: 16,
            color: AppColors.statusActive,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              'Answered: ${askUser.answer ?? ''}',
              style: theme.textTheme.bodyMedium?.copyWith(
                fontStyle: FontStyle.italic,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

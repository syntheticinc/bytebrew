import 'package:flutter/material.dart';

import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';

/// Displays agent lifecycle events (spawned, completed, failed) as
/// compact, expandable blocks similar to tool call widgets.
class AgentLifecycleWidget extends StatefulWidget {
  const AgentLifecycleWidget({super.key, required this.message});

  final ChatMessage message;

  @override
  State<AgentLifecycleWidget> createState() => _AgentLifecycleWidgetState();
}

class _AgentLifecycleWidgetState extends State<AgentLifecycleWidget> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final parsed = _parseContent(widget.message.content);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: parsed.description.isNotEmpty
            ? () => setState(() => _expanded = !_expanded)
            : null,
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
              _buildHeader(parsed, theme),
              if (_expanded && parsed.description.isNotEmpty)
                _buildDescription(parsed.description, theme),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildHeader(_LifecycleInfo info, ThemeData theme) {
    final icon = switch (info.type) {
      _LifecycleType.spawned => Icon(
        Icons.play_circle_outline,
        size: 14,
        color: Colors.green.shade400,
      ),
      _LifecycleType.completed => Icon(
        Icons.check_circle_outline,
        size: 14,
        color: Colors.green.shade400,
      ),
      _LifecycleType.failed => Icon(
        Icons.cancel_outlined,
        size: 14,
        color: theme.colorScheme.error,
      ),
    };

    final verb = switch (info.type) {
      _LifecycleType.spawned => 'started',
      _LifecycleType.completed => 'completed',
      _LifecycleType.failed => 'failed',
    };

    return Row(
      children: [
        icon,
        const SizedBox(width: 6),
        Text(
          'Agent $verb',
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w500,
          ),
        ),
        if (info.agentLabel.isNotEmpty) ...[
          const SizedBox(width: 4),
          Flexible(
            child: Text(
              info.agentLabel,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: AppColors.shade3,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
        if (info.description.isNotEmpty) ...[
          const Spacer(),
          Icon(
            _expanded ? Icons.expand_less : Icons.expand_more,
            size: 16,
            color: AppColors.shade3,
          ),
        ],
      ],
    );
  }

  Widget _buildDescription(String description, ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.only(left: 20, top: 4),
      child: Text(
        description,
        style: theme.textTheme.bodySmall?.copyWith(color: AppColors.shade3),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

enum _LifecycleType { spawned, completed, failed }

class _LifecycleInfo {
  const _LifecycleInfo({
    required this.type,
    required this.agentLabel,
    required this.description,
  });

  final _LifecycleType type;
  final String agentLabel;
  final String description;
}

/// Parses "Agent started: `<description>`" content into structured info.
_LifecycleInfo _parseContent(String content) {
  final type = content.contains('started')
      ? _LifecycleType.spawned
      : content.contains('completed')
      ? _LifecycleType.completed
      : _LifecycleType.failed;

  // Extract the part after "Agent started: " etc.
  final colonIndex = content.indexOf(': ');
  if (colonIndex == -1) {
    return _LifecycleInfo(type: type, agentLabel: '', description: '');
  }

  final rawDescription = content.substring(colonIndex + 2).trim();

  // The description may contain the agent name as first line/word,
  // and task description as the rest. Split on first newline if present.
  final newlineIndex = rawDescription.indexOf('\n');
  if (newlineIndex != -1) {
    return _LifecycleInfo(
      type: type,
      agentLabel: rawDescription.substring(0, newlineIndex).trim(),
      description: rawDescription.substring(newlineIndex + 1).trim(),
    );
  }

  // Single line — use as label, no description to expand.
  return _LifecycleInfo(
    type: type,
    agentLabel: rawDescription,
    description: '',
  );
}

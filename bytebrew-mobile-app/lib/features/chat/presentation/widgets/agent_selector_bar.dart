import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/domain/agent_info.dart';
import '../../application/agent_provider.dart';

/// Horizontal chip bar for switching between agent views.
///
/// Shows "Supervisor" + one chip per sub-agent. Hidden when no agents exist.
class AgentSelectorBar extends ConsumerWidget {
  const AgentSelectorBar({super.key, required this.sessionId});

  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final agents = ref.watch(agentsProvider(sessionId));
    if (agents.isEmpty) return const SizedBox.shrink();

    final selectedAgentId = ref.watch(selectedAgentProvider(sessionId));
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        border: Border(
          bottom: BorderSide(
            color: colorScheme.outlineVariant.withValues(alpha: 0.3),
          ),
        ),
      ),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          children: [
            _buildChip(
              context: context,
              label: 'Supervisor',
              isSelected: selectedAgentId == null,
              onSelected: () {
                ref
                    .read(selectedAgentProvider(sessionId).notifier)
                    .select(null);
              },
            ),
            ...agents.map(
              (agent) => Padding(
                padding: const EdgeInsets.only(left: 8),
                child: _buildChip(
                  context: context,
                  label: _shortLabel(agent.description),
                  isSelected: selectedAgentId == agent.agentId,
                  status: agent.status,
                  onSelected: () {
                    ref
                        .read(selectedAgentProvider(sessionId).notifier)
                        .select(agent.agentId);
                  },
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildChip({
    required BuildContext context,
    required String label,
    required bool isSelected,
    AgentStatus? status,
    required VoidCallback onSelected,
  }) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final statusIcon = switch (status) {
      AgentStatus.running => Icon(
        Icons.circle,
        size: 8,
        color: Colors.green.shade400,
      ),
      AgentStatus.completed => Icon(
        Icons.check_circle,
        size: 12,
        color: Colors.green.shade400,
      ),
      AgentStatus.failed => Icon(
        Icons.error,
        size: 12,
        color: colorScheme.error,
      ),
      null => null,
    };

    return FilterChip(
      label: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (statusIcon != null) ...[statusIcon, const SizedBox(width: 4)],
          Text(label),
        ],
      ),
      selected: isSelected,
      onSelected: (_) => onSelected(),
      showCheckmark: false,
      selectedColor: colorScheme.primaryContainer,
      backgroundColor: colorScheme.surfaceContainerHigh,
      labelStyle: theme.textTheme.labelSmall?.copyWith(
        color: isSelected
            ? colorScheme.onPrimaryContainer
            : colorScheme.onSurfaceVariant,
      ),
      padding: const EdgeInsets.symmetric(horizontal: 4),
      visualDensity: VisualDensity.compact,
    );
  }

  String _shortLabel(String description) {
    if (description.length <= 20) return description;
    return '${description.substring(0, 17)}...';
  }
}

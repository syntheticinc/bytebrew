/// Status of an agent in a multi-agent session.
enum AgentStatus { running, completed, failed }

/// Describes a sub-agent spawned during a chat session.
class AgentInfo {
  const AgentInfo({
    required this.agentId,
    required this.status,
    required this.description,
    required this.lastActivityAt,
  });

  final String agentId;
  final AgentStatus status;
  final String description;
  final DateTime lastActivityAt;

  AgentInfo copyWith({
    String? agentId,
    AgentStatus? status,
    String? description,
    DateTime? lastActivityAt,
  }) {
    return AgentInfo(
      agentId: agentId ?? this.agentId,
      status: status ?? this.status,
      description: description ?? this.description,
      lastActivityAt: lastActivityAt ?? this.lastActivityAt,
    );
  }
}

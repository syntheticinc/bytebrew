/// Status of an agent session.
enum SessionStatus { needsAttention, active, idle }

/// An active or recent agent session on a paired server.
class Session {
  const Session({
    required this.id,
    required this.serverId,
    required this.serverName,
    required this.projectName,
    required this.status,
    this.currentTask,
    required this.hasAskUser,
    required this.lastActivityAt,
  });

  final String id;
  final String serverId;
  final String serverName;
  final String projectName;
  final SessionStatus status;
  final String? currentTask;
  final bool hasAskUser;
  final DateTime lastActivityAt;

  Session copyWith({
    String? id,
    String? serverId,
    String? serverName,
    String? projectName,
    SessionStatus? status,
    String? currentTask,
    bool? hasAskUser,
    DateTime? lastActivityAt,
  }) {
    return Session(
      id: id ?? this.id,
      serverId: serverId ?? this.serverId,
      serverName: serverName ?? this.serverName,
      projectName: projectName ?? this.projectName,
      status: status ?? this.status,
      currentTask: currentTask ?? this.currentTask,
      hasAskUser: hasAskUser ?? this.hasAskUser,
      lastActivityAt: lastActivityAt ?? this.lastActivityAt,
    );
  }
}

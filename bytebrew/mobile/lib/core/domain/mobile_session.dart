/// Status of a mobile session on the CLI server.
enum MobileSessionState {
  unspecified,
  active,
  idle,
  needsAttention,
  completed,
  failed,
}

/// A session as reported by the CLI server's MobileService.
class MobileSession {
  const MobileSession({
    required this.sessionId,
    required this.projectKey,
    required this.projectRoot,
    required this.status,
    required this.currentTask,
    required this.startedAt,
    required this.lastActivityAt,
    required this.hasAskUser,
    required this.platform,
  });

  final String sessionId;
  final String projectKey;
  final String projectRoot;
  final MobileSessionState status;
  final String currentTask;
  final DateTime startedAt;
  final DateTime lastActivityAt;
  final bool hasAskUser;
  final String platform;

  /// Extracts a human-readable project name from [projectRoot].
  ///
  /// Splits on both `/` and `\` to support Unix and Windows paths.
  /// Falls back to [projectKey] if [projectRoot] is empty or contains
  /// only separators.
  String get projectName {
    if (projectRoot.isEmpty) return projectKey;

    final segments = projectRoot
        .split(RegExp(r'[/\\]'))
        .where((s) => s.isNotEmpty)
        .toList();

    if (segments.isEmpty) return projectKey;
    return segments.last;
  }
}

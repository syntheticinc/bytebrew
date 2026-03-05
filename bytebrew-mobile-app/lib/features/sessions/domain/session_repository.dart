import 'package:bytebrew_mobile/core/domain/session.dart';

/// Repository for listing and refreshing agent sessions.
abstract class SessionRepository {
  /// Returns all known sessions across paired servers.
  Future<List<Session>> listSessions();

  /// Forces a refresh of session data from servers.
  Future<void> refresh();

  /// Stream of session list updates. Returns null if not supported
  /// (e.g. [EmptySessionRepository]).
  Stream<List<Session>>? watchSessions();
}

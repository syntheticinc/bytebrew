import 'package:flutter/foundation.dart';

import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/features/sessions/domain/session_repository.dart';

/// [SessionRepository] implementation backed by WebSocket via
/// [WsConnectionManager].
///
/// Gathers sessions from all actively connected servers, maps them to the
/// domain [Session] model, and sorts them by priority (status) then recency.
class WsSessionRepository implements SessionRepository {
  WsSessionRepository({required WsConnectionManager connectionManager})
    : _connectionManager = connectionManager;

  final WsConnectionManager _connectionManager;

  @override
  Future<List<Session>> listSessions() async {
    final sessions = <Session>[];

    for (final connection in _connectionManager.activeConnections) {
      try {
        final result = await connection.client.listSessions(
          deviceToken: connection.server.deviceToken!,
        );

        for (final mobileSess in result.sessions) {
          sessions.add(
            Session(
              id: mobileSess.sessionId,
              serverId: connection.server.id,
              serverName: result.serverName,
              projectName: mobileSess.projectName,
              status: _mapStatus(mobileSess.status),
              currentTask: mobileSess.currentTask.isEmpty
                  ? null
                  : mobileSess.currentTask,
              hasAskUser: mobileSess.hasAskUser,
              lastActivityAt: mobileSess.lastActivityAt,
            ),
          );
        }
      } catch (e) {
        debugPrint(
          '[Sessions] Failed to list sessions from '
          '${connection.server.name}: $e',
        );
      }
    }

    sessions.sort(_compareSessionPriority);
    return sessions;
  }

  @override
  Future<void> refresh() async {
    // No-op: listSessions always fetches fresh data from servers.
  }

  @override
  Stream<List<Session>>? watchSessions() => null;

  // ---------------------------------------------------------------------------
  // Mapping
  // ---------------------------------------------------------------------------

  static SessionStatus _mapStatus(MobileSessionState state) {
    return switch (state) {
      MobileSessionState.active => SessionStatus.active,
      MobileSessionState.idle => SessionStatus.idle,
      MobileSessionState.needsAttention => SessionStatus.needsAttention,
      MobileSessionState.completed => SessionStatus.idle,
      MobileSessionState.failed => SessionStatus.idle,
      MobileSessionState.unspecified => SessionStatus.idle,
    };
  }

  // ---------------------------------------------------------------------------
  // Sorting
  // ---------------------------------------------------------------------------

  /// Priority order: needsAttention (0) > active (1) > idle (2).
  static int _statusPriority(SessionStatus status) {
    return switch (status) {
      SessionStatus.needsAttention => 0,
      SessionStatus.active => 1,
      SessionStatus.idle => 2,
    };
  }

  /// Sorts sessions by status priority (ascending) then by
  /// [Session.lastActivityAt] descending (most recent first).
  static int _compareSessionPriority(Session a, Session b) {
    final priorityDiff = _statusPriority(a.status) - _statusPriority(b.status);
    if (priorityDiff != 0) return priorityDiff;

    // Within same status group, most recent first.
    return b.lastActivityAt.compareTo(a.lastActivityAt);
  }
}

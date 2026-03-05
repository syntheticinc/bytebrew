import 'dart:async';

import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/features/sessions/domain/session_repository.dart';

/// [SessionRepository] backed by a WebSocket connection to the CLI.
///
/// Builds a single [Session] from the WS connection metadata and updates
/// its status in response to processing/ask-user events.
class WsSessionRepository implements SessionRepository {
  WsSessionRepository({required WsConnection connection})
    : _connection = connection {
    _eventSub = _connection.events.listen(_handleEvent);
    _emitSession();
  }

  final WsConnection _connection;
  final _controller = StreamController<List<Session>>.broadcast();
  StreamSubscription<Map<String, dynamic>>? _eventSub;

  SessionStatus _status = SessionStatus.idle;
  bool _hasAskUser = false;

  @override
  Future<List<Session>> listSessions() async => [_buildSession()];

  @override
  Future<void> refresh() async => _emitSession();

  @override
  Stream<List<Session>> watchSessions() => _controller.stream;

  /// Releases resources. Call when the repository is no longer needed.
  void dispose() {
    _eventSub?.cancel();
    _controller.close();
  }

  void _handleEvent(Map<String, dynamic> event) {
    final type = event['type'] as String?;
    var changed = false;

    if (type == 'ProcessingStarted') {
      _status = SessionStatus.active;
      changed = true;
    }
    if (type == 'ProcessingStopped') {
      _status = SessionStatus.idle;
      changed = true;
    }
    if (type == 'AskUserRequested') {
      _hasAskUser = true;
      _status = SessionStatus.needsAttention;
      changed = true;
    }
    if (type == 'AskUserResolved') {
      _hasAskUser = false;
      _status = SessionStatus.active;
      changed = true;
    }

    if (changed) _emitSession();
  }

  Session _buildSession() => Session(
    id: _connection.meta['sessionId'] as String? ?? 'cli-session',
    serverId: 'cli',
    serverName: _connection.meta['projectName'] as String? ?? 'CLI',
    projectName: _connection.meta['projectName'] as String? ?? '',
    status: _status,
    hasAskUser: _hasAskUser,
    lastActivityAt: DateTime.now(),
  );

  void _emitSession() {
    if (_controller.isClosed) return;
    _controller.add([_buildSession()]);
  }
}

import 'dart:async';

import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/sessions/application/auto_connect_provider.dart';
import 'package:bytebrew_mobile/features/sessions/domain/session_repository.dart';
import 'package:bytebrew_mobile/features/sessions/infrastructure/ws_session_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'sessions_provider.g.dart';

/// Provides the [SessionRepository] backed by WS via [WsConnectionManager].
@Riverpod(keepAlive: true)
SessionRepository sessionRepository(Ref ref) {
  final manager = ref.read(connectionManagerProvider);
  return WsSessionRepository(connectionManager: manager);
}

/// Manages the list of agent sessions.
///
/// Subscribes to push-based session updates from the server via
/// [SessionRepository.watchSessions]. No polling is needed.
@riverpod
class Sessions extends _$Sessions {
  StreamSubscription<List<Session>>? _streamSub;

  @override
  FutureOr<List<Session>> build() async {
    // Wait for auto-connect to complete before fetching sessions.
    await ref.watch(sessionsAutoConnectProvider.future);

    final repo = ref.watch(sessionRepositoryProvider);

    // Cancel previous subscription on rebuild.
    _streamSub?.cancel();
    ref.onDispose(() => _streamSub?.cancel());

    // Subscribe to push updates.
    final stream = repo.watchSessions();
    if (stream != null) {
      _streamSub = stream.listen((sessions) {
        state = AsyncData(sessions);
      });
    }

    return repo.listSessions();
  }

  /// Forces a refresh of session data from servers.
  Future<void> refresh() async {
    // Keep current data visible while refreshing (no loading flash).
    final previous = state.value ?? [];
    state = AsyncData(previous);
    try {
      final repo = ref.read(sessionRepositoryProvider);
      await repo.refresh();
      state = AsyncData(await repo.listSessions());
    } on Exception catch (e, st) {
      state = AsyncError(e, st);
    }
  }
}

/// Groups sessions by their [SessionStatus].
@riverpod
Map<SessionStatus, List<Session>> groupedSessions(Ref ref) {
  final sessionsAsync = ref.watch(sessionsProvider);
  return sessionsAsync.when(
    data: (sessions) {
      final grouped = <SessionStatus, List<Session>>{};
      for (final status in SessionStatus.values) {
        final filtered = sessions.where((s) => s.status == status).toList();
        if (filtered.isNotEmpty) {
          grouped[status] = filtered;
        }
      }
      return grouped;
    },
    loading: () => {},
    error: (_, _) => {},
  );
}

/// Finds a single session by [id], or null if not found.
@riverpod
Session? sessionById(Ref ref, String id) {
  final sessionsAsync = ref.watch(sessionsProvider);
  return sessionsAsync.whenOrNull(
    data: (sessions) => sessions.where((s) => s.id == id).firstOrNull,
  );
}

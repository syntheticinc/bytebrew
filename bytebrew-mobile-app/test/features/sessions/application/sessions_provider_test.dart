import 'dart:async';

import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/sessions/domain/session_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../helpers/fakes.dart';

// ---------------------------------------------------------------------------
// Fake session repository
// ---------------------------------------------------------------------------

class _FakeSessionRepository implements SessionRepository {
  _FakeSessionRepository(this._sessions);

  final List<Session> _sessions;
  int refreshCount = 0;

  @override
  Future<List<Session>> listSessions() async => _sessions;

  @override
  Future<void> refresh() async {
    refreshCount++;
  }
}

class _FailingSessionRepository implements SessionRepository {
  @override
  Future<List<Session>> listSessions() async {
    throw Exception('Network error');
  }

  @override
  Future<void> refresh() async {}
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

final _now = DateTime.now();

final _testSessions = [
  Session(
    id: 'session-1',
    serverId: 'srv-1',
    serverName: 'MacBook Pro',
    projectName: 'api-gateway',
    status: SessionStatus.needsAttention,
    hasAskUser: true,
    lastActivityAt: _now.subtract(const Duration(minutes: 1)),
  ),
  Session(
    id: 'session-2',
    serverId: 'srv-1',
    serverName: 'MacBook Pro',
    projectName: 'bytebrew-srv',
    status: SessionStatus.active,
    currentTask: 'Refactoring auth',
    hasAskUser: false,
    lastActivityAt: _now.subtract(const Duration(minutes: 2)),
  ),
  Session(
    id: 'session-3',
    serverId: 'srv-2',
    serverName: 'Desktop PC',
    projectName: 'test-project',
    status: SessionStatus.idle,
    hasAskUser: false,
    lastActivityAt: _now.subtract(const Duration(minutes: 15)),
  ),
  Session(
    id: 'session-4',
    serverId: 'srv-2',
    serverName: 'Desktop PC',
    projectName: 'mobile-app',
    status: SessionStatus.active,
    hasAskUser: false,
    lastActivityAt: _now.subtract(const Duration(hours: 2)),
  ),
];

void main() {
  // =========================================================================
  // Sessions (AsyncNotifier)
  // =========================================================================
  group('Sessions', () {
    test('loads sessions from repository on build', () async {
      final fakeRepo = _FakeSessionRepository(_testSessions);

      final container = ProviderContainer(
        overrides: [
          sessionRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      final sessions = await container.read(sessionsProvider.future);

      expect(sessions, hasLength(4));
      expect(sessions.first.projectName, 'api-gateway');
    });

    test('returns empty list from empty repository', () async {
      final fakeRepo = _FakeSessionRepository([]);

      final container = ProviderContainer(
        overrides: [
          sessionRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      final sessions = await container.read(sessionsProvider.future);
      expect(sessions, isEmpty);
    });

    test('refresh calls repository refresh and reloads data', () async {
      final fakeRepo = _FakeSessionRepository(_testSessions);

      final container = ProviderContainer(
        overrides: [
          sessionRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      // Initial load.
      await container.read(sessionsProvider.future);

      // Refresh.
      await container.read(sessionsProvider.notifier).refresh();

      expect(fakeRepo.refreshCount, 1);

      final sessions = await container.read(sessionsProvider.future);
      expect(sessions, hasLength(4));
    });

    // Note: error handling in build() is delegated to Riverpod's AsyncNotifier.
    // Testing that Riverpod transitions to AsyncError is testing framework
    // internals, not our business logic, so it's omitted here.
  });

  // =========================================================================
  // groupedSessionsProvider
  // =========================================================================
  group('groupedSessionsProvider', () {
    test('groups sessions by status', () async {
      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier(_testSessions),
          ),
        ],
      );
      addTearDown(container.dispose);

      // Wait for sessions to load.
      await container.read(sessionsProvider.future);

      final grouped = container.read(groupedSessionsProvider);

      expect(grouped[SessionStatus.needsAttention], hasLength(1));
      expect(grouped[SessionStatus.active], hasLength(2));
      expect(grouped[SessionStatus.idle], hasLength(1));
    });

    test('returns empty map when no sessions', () async {
      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
        ],
      );
      addTearDown(container.dispose);

      await container.read(sessionsProvider.future);

      final grouped = container.read(groupedSessionsProvider);
      expect(grouped, isEmpty);
    });

    test('omits status groups with zero sessions', () async {
      final onlyActive = [
        Session(
          id: 'session-x',
          serverId: 'srv-1',
          serverName: 'Server',
          projectName: 'project',
          status: SessionStatus.active,
          hasAskUser: false,
          lastActivityAt: _now,
        ),
      ];

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier(onlyActive),
          ),
        ],
      );
      addTearDown(container.dispose);

      await container.read(sessionsProvider.future);

      final grouped = container.read(groupedSessionsProvider);
      expect(grouped.containsKey(SessionStatus.active), isTrue);
      expect(grouped.containsKey(SessionStatus.idle), isFalse);
      expect(grouped.containsKey(SessionStatus.needsAttention), isFalse);
    });
  });

  // =========================================================================
  // sessionByIdProvider
  // =========================================================================
  group('sessionByIdProvider', () {
    test('returns session when found', () async {
      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier(_testSessions),
          ),
        ],
      );
      addTearDown(container.dispose);

      await container.read(sessionsProvider.future);

      final session = container.read(sessionByIdProvider('session-2'));
      expect(session, isNotNull);
      expect(session!.projectName, 'bytebrew-srv');
    });

    test('returns null when session not found', () async {
      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier(_testSessions),
          ),
        ],
      );
      addTearDown(container.dispose);

      await container.read(sessionsProvider.future);

      final session = container.read(sessionByIdProvider('nonexistent'));
      expect(session, isNull);
    });
  });
}

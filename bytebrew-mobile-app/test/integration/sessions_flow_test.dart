import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/chat/application/connection_provider.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/sessions_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../helpers/fakes.dart';

final _testSessions = [
  Session(
    id: 'test-1',
    serverId: 'srv-1',
    serverName: 'MacBook Pro',
    projectName: 'api-gateway',
    status: SessionStatus.active,
    currentTask: 'Running tests',
    hasAskUser: false,
    lastActivityAt: DateTime.now().subtract(const Duration(minutes: 1)),
  ),
  Session(
    id: 'test-2',
    serverId: 'srv-2',
    serverName: 'Desktop PC',
    projectName: 'web-app',
    status: SessionStatus.needsAttention,
    hasAskUser: true,
    lastActivityAt: DateTime.now().subtract(const Duration(minutes: 5)),
  ),
  Session(
    id: 'test-3',
    serverId: 'srv-1',
    serverName: 'MacBook Pro',
    projectName: 'old-project',
    status: SessionStatus.idle,
    hasAskUser: false,
    lastActivityAt: DateTime.now().subtract(const Duration(hours: 2)),
  ),
];

final _testGrouped = <SessionStatus, List<Session>>{
  SessionStatus.needsAttention: [_testSessions[1]],
  SessionStatus.active: [_testSessions[0]],
  SessionStatus.idle: [_testSessions[2]],
};

Widget _buildSessionsScreen({
  List<Session>? sessions,
  Map<SessionStatus, List<Session>>? grouped,
  WsConnectionStatus wsStatus = WsConnectionStatus.disconnected,
}) {
  return ProviderScope(
    overrides: [
      sessionsProvider.overrideWith(() => FakeSessionsNotifier(sessions ?? [])),
      groupedSessionsProvider.overrideWithValue(grouped ?? {}),
      wsConnectionProvider.overrideWith(() => FakeWsConnection(wsStatus)),
    ],
    child: const MaterialApp(home: SessionsScreen()),
  );
}

void main() {
  group('Sessions flow integration', () {
    testWidgets('TC-SESS-01: Grouped sessions display headers and cards', (
      tester,
    ) async {
      await tester.pumpWidget(
        _buildSessionsScreen(sessions: _testSessions, grouped: _testGrouped),
      );

      // Use pump instead of pumpAndSettle because animated status
      // indicators have infinite repeating animations.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Verify group headers with counts.
      expect(find.text('ACTION REQUIRED (1)'), findsOneWidget);
      expect(find.text('IN PROGRESS (1)'), findsOneWidget);
      expect(find.text('RECENT (1)'), findsOneWidget);

      // Verify session project names are displayed.
      expect(find.text('api-gateway'), findsOneWidget);
      expect(find.text('web-app'), findsOneWidget);
      expect(find.text('old-project'), findsOneWidget);
    });

    testWidgets('TC-SESS-02: Empty state shows placeholder message', (
      tester,
    ) async {
      await tester.pumpWidget(_buildSessionsScreen(sessions: [], grouped: {}));
      await tester.pumpAndSettle();

      expect(find.text('No sessions yet'), findsOneWidget);
      expect(find.text('Your agent sessions will appear here'), findsOneWidget);
    });

    testWidgets('TC-SESS-03: Live Session card appears when WS is connected', (
      tester,
    ) async {
      await tester.pumpWidget(
        _buildSessionsScreen(
          sessions: _testSessions,
          grouped: _testGrouped,
          wsStatus: WsConnectionStatus.connected,
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Live Session card should be visible.
      expect(find.text('Live Session'), findsOneWidget);
      expect(find.text('Connected to CLI'), findsOneWidget);
    });

    testWidgets('TC-SESS-04: Session details display correctly', (
      tester,
    ) async {
      await tester.pumpWidget(
        _buildSessionsScreen(sessions: _testSessions, grouped: _testGrouped),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Active session shows current task.
      expect(find.text('Running tests'), findsOneWidget);

      // Server names are displayed.
      expect(find.text('MacBook Pro'), findsWidgets);
      expect(find.text('Desktop PC'), findsOneWidget);
    });

    testWidgets('TC-SESS-05: Pull to refresh does not crash', (tester) async {
      await tester.pumpWidget(
        _buildSessionsScreen(sessions: _testSessions, grouped: _testGrouped),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Perform a drag down gesture to trigger RefreshIndicator.
      // Find the scrollable area and drag from top.
      await tester.fling(find.byType(ListView), const Offset(0, 300), 1000);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));

      // Should not crash and content should still be visible.
      expect(find.text('api-gateway'), findsOneWidget);
    });
  });
}

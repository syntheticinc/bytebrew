import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/sessions/application/auto_connect_provider.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/sessions_screen.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../helpers/fakes.dart';

void main() {
  final testSessions = [
    Session(
      id: 'test-1',
      serverId: 'srv-1',
      serverName: 'Test Server',
      projectName: 'api-gateway',
      status: SessionStatus.active,
      currentTask: 'Running tests',
      hasAskUser: false,
      lastActivityAt: DateTime.now().subtract(const Duration(minutes: 1)),
    ),
    Session(
      id: 'test-2',
      serverId: 'srv-1',
      serverName: 'Test Server',
      projectName: 'web-app',
      status: SessionStatus.needsAttention,
      hasAskUser: true,
      lastActivityAt: DateTime.now().subtract(const Duration(minutes: 5)),
    ),
    Session(
      id: 'test-3',
      serverId: 'srv-1',
      serverName: 'Test Server',
      projectName: 'old-project',
      status: SessionStatus.idle,
      hasAskUser: false,
      lastActivityAt: DateTime.now().subtract(const Duration(hours: 2)),
    ),
  ];

  final testGrouped = <SessionStatus, List<Session>>{
    SessionStatus.needsAttention: [testSessions[1]],
    SessionStatus.active: [testSessions[0]],
    SessionStatus.idle: [testSessions[2]],
  };

  testWidgets('SessionsScreen renders group headers when sessions exist', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier(testSessions),
          ),
          groupedSessionsProvider.overrideWithValue(testGrouped),
          settingsRepositoryProvider.overrideWithValue(
            FakeSettingsRepository(),
          ),
          sessionsAutoConnectProvider.overrideWith((ref) async {}),
          connectionManagerProvider.overrideWithValue(
            FakeConnectionManager(),
          ),
          serversProvider.overrideWithValue([]),
        ],
        child: const MaterialApp(home: SessionsScreen()),
      ),
    );

    // Let async provider resolve (use pump instead of pumpAndSettle because
    // AnimatedStatusIndicator has an infinite repeating animation for
    // needsAttention status)
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    // Verify group headers are displayed with counts
    expect(find.text('ACTION REQUIRED (1)'), findsOneWidget);
    expect(find.text('IN PROGRESS (1)'), findsOneWidget);
    expect(find.text('RECENT (1)'), findsOneWidget);

    // Verify session project names are displayed
    expect(find.text('api-gateway'), findsOneWidget);
    expect(find.text('web-app'), findsOneWidget);
    expect(find.text('old-project'), findsOneWidget);
  });

  testWidgets('SessionsScreen shows empty state when no sessions', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
          groupedSessionsProvider.overrideWithValue({}),
          settingsRepositoryProvider.overrideWithValue(
            FakeSettingsRepository(),
          ),
          sessionsAutoConnectProvider.overrideWith((ref) async {}),
          connectionManagerProvider.overrideWithValue(
            FakeConnectionManager(),
          ),
          serversProvider.overrideWithValue([]),
        ],
        child: const MaterialApp(home: SessionsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('No sessions yet'), findsOneWidget);
    expect(find.text('Your agent sessions will appear here'), findsOneWidget);
  });

  testWidgets('SessionsScreen shows AppBar with Activity title', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
          groupedSessionsProvider.overrideWithValue({}),
          settingsRepositoryProvider.overrideWithValue(
            FakeSettingsRepository(),
          ),
          sessionsAutoConnectProvider.overrideWith((ref) async {}),
          connectionManagerProvider.overrideWithValue(
            FakeConnectionManager(),
          ),
          serversProvider.overrideWithValue([]),
        ],
        child: const MaterialApp(home: SessionsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Activity'), findsOneWidget);
  });
}

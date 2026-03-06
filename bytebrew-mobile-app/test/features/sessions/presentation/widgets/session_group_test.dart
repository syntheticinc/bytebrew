import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/widgets/session_card.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/widgets/session_group.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SessionGroup', () {
    final activeSessions = [
      Session(
        id: 'sess-1',
        serverId: 'srv-1',
        serverName: 'Production',
        projectName: 'api-gateway',
        status: SessionStatus.active,
        currentTask: 'Running tests',
        hasAskUser: false,
        lastActivityAt: DateTime.now().subtract(const Duration(minutes: 1)),
      ),
      Session(
        id: 'sess-2',
        serverId: 'srv-1',
        serverName: 'Production',
        projectName: 'web-app',
        status: SessionStatus.active,
        hasAskUser: false,
        lastActivityAt: DateTime.now().subtract(const Duration(minutes: 3)),
      ),
    ];

    final attentionSessions = [
      Session(
        id: 'sess-3',
        serverId: 'srv-1',
        serverName: 'Staging',
        projectName: 'auth-service',
        status: SessionStatus.needsAttention,
        hasAskUser: true,
        lastActivityAt: DateTime.now().subtract(const Duration(minutes: 5)),
      ),
    ];

    final idleSessions = [
      Session(
        id: 'sess-4',
        serverId: 'srv-1',
        serverName: 'Dev',
        projectName: 'old-project',
        status: SessionStatus.idle,
        hasAskUser: false,
        lastActivityAt: DateTime.now().subtract(const Duration(hours: 2)),
      ),
    ];

    Widget buildWidget(SessionStatus status, List<Session> sessions) {
      return MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: SessionGroup(status: status, sessions: sessions),
          ),
        ),
      );
    }

    testWidgets('renders "IN PROGRESS" header with count for active status', (
      tester,
    ) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.active, activeSessions),
      );
      // Use pump() because AnimatedStatusIndicator animates.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('IN PROGRESS (2)'), findsOneWidget);
    });

    testWidgets('renders "ACTION REQUIRED" header for needsAttention status', (
      tester,
    ) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.needsAttention, attentionSessions),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('ACTION REQUIRED (1)'), findsOneWidget);
    });

    testWidgets('renders "RECENT" header for idle status', (tester) async {
      await tester.pumpWidget(buildWidget(SessionStatus.idle, idleSessions));
      await tester.pumpAndSettle();

      expect(find.text('RECENT (1)'), findsOneWidget);
    });

    testWidgets('header color is green for active status', (tester) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.active, activeSessions),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final headerText = tester.widget<Text>(find.text('IN PROGRESS (2)'));
      expect(headerText.style?.color, AppColors.statusActive);
    });

    testWidgets('header color is accent for needsAttention status', (
      tester,
    ) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.needsAttention, attentionSessions),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final headerText = tester.widget<Text>(find.text('ACTION REQUIRED (1)'));
      expect(headerText.style?.color, AppColors.accent);
    });

    testWidgets('header color is shade3 for idle status', (tester) async {
      await tester.pumpWidget(buildWidget(SessionStatus.idle, idleSessions));
      await tester.pumpAndSettle();

      final headerText = tester.widget<Text>(find.text('RECENT (1)'));
      expect(headerText.style?.color, AppColors.shade3);
    });

    testWidgets('renders SessionCard for each session', (tester) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.active, activeSessions),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Should render 2 SessionCard widgets.
      expect(find.byType(SessionCard), findsNWidgets(2));
    });

    testWidgets('renders session project names', (tester) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.active, activeSessions),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('api-gateway'), findsOneWidget);
      expect(find.text('web-app'), findsOneWidget);
    });

    testWidgets('renders single session correctly', (tester) async {
      await tester.pumpWidget(
        buildWidget(SessionStatus.needsAttention, attentionSessions),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.byType(SessionCard), findsOneWidget);
      expect(find.text('auth-service'), findsOneWidget);
    });

    testWidgets('header has letter spacing', (tester) async {
      await tester.pumpWidget(buildWidget(SessionStatus.idle, idleSessions));
      await tester.pumpAndSettle();

      final headerText = tester.widget<Text>(find.text('RECENT (1)'));
      expect(headerText.style?.letterSpacing, 2);
    });

    testWidgets('header has bold font weight', (tester) async {
      await tester.pumpWidget(buildWidget(SessionStatus.idle, idleSessions));
      await tester.pumpAndSettle();

      final headerText = tester.widget<Text>(find.text('RECENT (1)'));
      expect(headerText.style?.fontWeight, FontWeight.w600);
    });

    testWidgets('renders in dark theme without errors', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData.dark(),
          home: Scaffold(
            body: SingleChildScrollView(
              child: SessionGroup(
                status: SessionStatus.active,
                sessions: activeSessions,
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('IN PROGRESS (2)'), findsOneWidget);
    });
  });
}

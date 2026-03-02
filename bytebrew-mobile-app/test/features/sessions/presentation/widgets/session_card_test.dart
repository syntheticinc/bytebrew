import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/core/widgets/animated_status_indicator.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/widgets/session_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SessionCard', () {
    Widget buildWidget(Session session) {
      return MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(child: SessionCard(session: session)),
        ),
      );
    }

    final activeSession = Session(
      id: 'sess-1',
      serverId: 'srv-1',
      serverName: 'Production',
      projectName: 'api-gateway',
      status: SessionStatus.active,
      currentTask: 'Running tests',
      hasAskUser: false,
      lastActivityAt: DateTime.now().subtract(const Duration(minutes: 2)),
    );

    final needsAttentionSession = Session(
      id: 'sess-2',
      serverId: 'srv-1',
      serverName: 'Staging',
      projectName: 'web-app',
      status: SessionStatus.needsAttention,
      hasAskUser: true,
      lastActivityAt: DateTime.now().subtract(const Duration(minutes: 5)),
    );

    final idleSession = Session(
      id: 'sess-3',
      serverId: 'srv-1',
      serverName: 'Dev',
      projectName: 'old-project',
      status: SessionStatus.idle,
      hasAskUser: false,
      lastActivityAt: DateTime.now().subtract(const Duration(hours: 3)),
    );

    testWidgets('renders project name', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      // Use pump() instead of pumpAndSettle() because AnimatedStatusIndicator
      // may have repeating animations for non-idle statuses.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('api-gateway'), findsOneWidget);
    });

    testWidgets('renders server name', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Production'), findsOneWidget);
    });

    testWidgets('renders current task when present', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Running tests'), findsOneWidget);
    });

    testWidgets('does not render current task when null', (tester) async {
      await tester.pumpWidget(buildWidget(idleSession));
      await tester.pumpAndSettle();

      // idleSession has no currentTask.
      expect(find.text('Running tests'), findsNothing);
    });

    testWidgets('shows "Waiting" badge when hasAskUser is true', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget(needsAttentionSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Waiting'), findsOneWidget);
    });

    testWidgets('does not show "Waiting" badge when hasAskUser is false', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Waiting'), findsNothing);
    });

    testWidgets('renders AnimatedStatusIndicator', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();

      expect(find.byType(AnimatedStatusIndicator), findsOneWidget);
    });

    testWidgets('renders time ago text', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Should show "2m ago" (approximately).
      expect(find.textContaining('ago'), findsOneWidget);
    });

    testWidgets('needsAttention status has accent left border', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget(needsAttentionSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Find the inner container with the left border decoration.
      final containers = tester.widgetList<Container>(
        find.byType(Container),
      );
      final borderedContainer = containers.where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.border != null) {
          final border = decoration.border;
          if (border is Border) {
            return border.left.color == AppColors.accent &&
                border.left.width == 2;
          }
        }
        return false;
      });
      expect(borderedContainer, isNotEmpty);
    });

    testWidgets('active status has green left border', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final containers = tester.widgetList<Container>(
        find.byType(Container),
      );
      final borderedContainer = containers.where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.border != null) {
          final border = decoration.border;
          if (border is Border) {
            return border.left.color == AppColors.statusActive &&
                border.left.width == 2;
          }
        }
        return false;
      });
      expect(borderedContainer, isNotEmpty);
    });

    testWidgets('idle status has no left border', (tester) async {
      await tester.pumpWidget(buildWidget(idleSession));
      await tester.pumpAndSettle();

      // Idle status uses BorderSide.none for the left border.
      // Verify no container has a 2px wide left border.
      final containers = tester.widgetList<Container>(
        find.byType(Container),
      );
      final borderedContainer = containers.where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.border != null) {
          final border = decoration.border;
          if (border is Border) {
            return border.left.width == 2;
          }
        }
        return false;
      });
      expect(borderedContainer, isEmpty);
    });

    testWidgets('"Waiting" badge has accent styling', (tester) async {
      await tester.pumpWidget(buildWidget(needsAttentionSession));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final waitingText = tester.widget<Text>(find.text('Waiting'));
      expect(waitingText.style?.color, AppColors.accent);
    });

    testWidgets('has InkWell for tap navigation', (tester) async {
      await tester.pumpWidget(buildWidget(activeSession));
      await tester.pump();

      expect(find.byType(InkWell), findsOneWidget);
    });

    testWidgets('renders in dark theme without errors', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData.dark(),
          home: Scaffold(
            body: SingleChildScrollView(
              child: SessionCard(session: activeSession),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('api-gateway'), findsOneWidget);
    });
  });
}

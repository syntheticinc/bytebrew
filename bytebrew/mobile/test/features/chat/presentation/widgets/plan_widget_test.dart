import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/plan_widget.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('PlanWidget', () {
    final planWithMixedSteps = PlanData(
      goal: 'Refactor auth module',
      steps: [
        PlanStep(
          index: 0,
          description: 'Extract TokenService',
          status: PlanStepStatus.completed,
          completedAt: DateTime(2026, 3, 1, 12, 0),
        ),
        const PlanStep(
          index: 1,
          description: 'Create AuthUsecase',
          status: PlanStepStatus.inProgress,
        ),
        const PlanStep(
          index: 2,
          description: 'Write tests',
          status: PlanStepStatus.pending,
        ),
      ],
    );

    ChatMessage buildPlanMessage({PlanData? plan}) {
      return ChatMessage(
        id: 'plan-msg-1',
        type: ChatMessageType.planUpdate,
        content: '',
        timestamp: DateTime.now(),
        plan: plan,
      );
    }

    Widget buildWidget(ChatMessage message, {String sessionId = 'test-sess'}) {
      return MaterialApp(
        home: Scaffold(
          body: PlanWidget(message: message, sessionId: sessionId),
        ),
      );
    }

    testWidgets('renders SizedBox.shrink when plan is null', (tester) async {
      final message = buildPlanMessage(plan: null);
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // PlanWidget returns SizedBox.shrink() when plan is null.
      final sizedBox = tester.widget<SizedBox>(find.byType(SizedBox).first);
      expect(sizedBox.width, 0.0);
      expect(sizedBox.height, 0.0);
    });

    testWidgets('renders goal text', (tester) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      // Use pump() because CircularProgressIndicator animates indefinitely.
      await tester.pump();

      expect(find.text('Refactor auth module'), findsOneWidget);
    });

    testWidgets('renders progress percentage', (tester) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      // 1 out of 3 completed = 33%.
      expect(find.text('33% complete'), findsOneWidget);
    });

    testWidgets('renders 100% when all steps completed', (tester) async {
      final allDone = PlanData(
        goal: 'Done plan',
        steps: [
          PlanStep(
            index: 0,
            description: 'Step A',
            status: PlanStepStatus.completed,
            completedAt: DateTime(2026, 3, 1),
          ),
          PlanStep(
            index: 1,
            description: 'Step B',
            status: PlanStepStatus.completed,
            completedAt: DateTime(2026, 3, 1),
          ),
        ],
      );

      await tester.pumpWidget(buildWidget(buildPlanMessage(plan: allDone)));
      await tester.pumpAndSettle();

      expect(find.text('100% complete'), findsOneWidget);
    });

    testWidgets('renders step descriptions', (tester) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      expect(find.text('Extract TokenService'), findsOneWidget);
      expect(find.text('Create AuthUsecase'), findsOneWidget);
      expect(find.text('Write tests'), findsOneWidget);
    });

    testWidgets('renders check_circle icon for completed steps', (
      tester,
    ) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      // Completed step has a check_circle icon.
      final checkIcons = tester.widgetList<Icon>(
        find.byIcon(Icons.check_circle),
      );
      // At least one check_circle for the completed step.
      expect(checkIcons.length, greaterThanOrEqualTo(1));

      // Verify the check_circle icon has the green statusActive color.
      final completedIcon = checkIcons.firstWhere(
        (icon) => icon.size == 16 && icon.color == AppColors.statusActive,
      );
      expect(completedIcon, isNotNull);
    });

    testWidgets('renders CircularProgressIndicator for in-progress steps', (
      tester,
    ) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      // In-progress step renders a CircularProgressIndicator with strokeWidth 2.
      final indicators = tester.widgetList<CircularProgressIndicator>(
        find.byType(CircularProgressIndicator),
      );
      // Should find at least one (the in-progress step indicator).
      expect(indicators, isNotEmpty);
    });

    testWidgets('renders progress bar', (tester) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      expect(find.byType(LinearProgressIndicator), findsOneWidget);

      final progressBar = tester.widget<LinearProgressIndicator>(
        find.byType(LinearProgressIndicator),
      );
      // 1 out of 3 completed ~ 0.333.
      expect(progressBar.value, closeTo(0.333, 0.01));
    });

    testWidgets('renders checklist icon in header', (tester) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      expect(find.byIcon(Icons.checklist), findsOneWidget);
    });

    testWidgets('renders chevron_right icon for navigation hint', (
      tester,
    ) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      expect(find.byIcon(Icons.chevron_right), findsOneWidget);
    });

    testWidgets('limits visible steps to 4 and shows "more steps" text', (
      tester,
    ) async {
      final manySteps = PlanData(
        goal: 'Big plan',
        steps: [
          for (var i = 0; i < 6; i++)
            PlanStep(
              index: i,
              description: 'Step $i',
              status: PlanStepStatus.pending,
            ),
        ],
      );

      await tester.pumpWidget(buildWidget(buildPlanMessage(plan: manySteps)));
      await tester.pumpAndSettle();

      // Should show first 4 steps.
      expect(find.text('Step 0'), findsOneWidget);
      expect(find.text('Step 1'), findsOneWidget);
      expect(find.text('Step 2'), findsOneWidget);
      expect(find.text('Step 3'), findsOneWidget);

      // Steps beyond the limit should not be shown.
      expect(find.text('Step 4'), findsNothing);
      expect(find.text('Step 5'), findsNothing);

      // Shows the "more steps" text.
      expect(find.text('2 more steps...'), findsOneWidget);
    });

    testWidgets('does not show "more steps" text when 4 or fewer steps', (
      tester,
    ) async {
      final message = buildPlanMessage(plan: planWithMixedSteps);
      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      // 3 steps total, no overflow text needed.
      expect(find.textContaining('more steps'), findsNothing);
    });
  });
}

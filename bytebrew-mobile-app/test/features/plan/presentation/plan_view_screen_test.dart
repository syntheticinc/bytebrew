import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/plan/presentation/plan_view_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('PlanViewScreen shows empty state when no plan', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(null)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.text('Plan'), findsOneWidget);
    expect(find.text('No active plan'), findsOneWidget);
    expect(
      find.text('The agent will create a plan when needed'),
      findsOneWidget,
    );
  });

  testWidgets('PlanViewScreen shows empty plan icon', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(null)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.byIcon(Icons.checklist_outlined), findsOneWidget);
  });

  testWidgets('PlanViewScreen renders plan goal and progress', (tester) async {
    const plan = PlanData(
      goal: 'Refactor authentication module',
      steps: [
        PlanStep(
          index: 0,
          description: 'Analyze current code',
          status: PlanStepStatus.completed,
        ),
        PlanStep(
          index: 1,
          description: 'Write unit tests',
          status: PlanStepStatus.inProgress,
        ),
        PlanStep(
          index: 2,
          description: 'Implement changes',
          status: PlanStepStatus.pending,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    // Goal is displayed in app bar subtitle and in content header.
    expect(find.text('Refactor authentication module'), findsWidgets);

    // Progress text: 1 of 3 completed = 33%.
    expect(find.text('33% complete  \u00b7  1/3 steps'), findsOneWidget);
  });

  testWidgets('PlanViewScreen renders step descriptions', (tester) async {
    const plan = PlanData(
      goal: 'Test goal',
      steps: [
        PlanStep(
          index: 0,
          description: 'Step one',
          status: PlanStepStatus.completed,
        ),
        PlanStep(
          index: 1,
          description: 'Step two',
          status: PlanStepStatus.inProgress,
        ),
        PlanStep(
          index: 2,
          description: 'Step three',
          status: PlanStepStatus.pending,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.text('Step one'), findsOneWidget);
    expect(find.text('Step two'), findsOneWidget);
    expect(find.text('Step three'), findsOneWidget);
  });

  testWidgets('PlanViewScreen shows check icon for completed steps', (
    tester,
  ) async {
    const plan = PlanData(
      goal: 'Goal',
      steps: [
        PlanStep(
          index: 0,
          description: 'Done step',
          status: PlanStepStatus.completed,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.byIcon(Icons.check_circle), findsOneWidget);
  });

  testWidgets('PlanViewScreen shows progress indicator for in-progress steps', (
    tester,
  ) async {
    const plan = PlanData(
      goal: 'Goal',
      steps: [
        PlanStep(
          index: 0,
          description: 'Active step',
          status: PlanStepStatus.inProgress,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('In progress...'), findsOneWidget);
  });

  testWidgets('PlanViewScreen shows circle icon for pending steps', (
    tester,
  ) async {
    const plan = PlanData(
      goal: 'Goal',
      steps: [
        PlanStep(
          index: 0,
          description: 'Pending step',
          status: PlanStepStatus.pending,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.byIcon(Icons.circle_outlined), findsOneWidget);
  });

  testWidgets('PlanViewScreen shows linear progress bar', (tester) async {
    const plan = PlanData(
      goal: 'Goal',
      steps: [
        PlanStep(
          index: 0,
          description: 'Step 1',
          status: PlanStepStatus.completed,
        ),
        PlanStep(
          index: 1,
          description: 'Step 2',
          status: PlanStepStatus.pending,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.byType(LinearProgressIndicator), findsOneWidget);
    expect(find.text('50% complete  \u00b7  1/2 steps'), findsOneWidget);
  });

  testWidgets('PlanViewScreen with all steps completed shows 100%', (
    tester,
  ) async {
    const plan = PlanData(
      goal: 'All done',
      steps: [
        PlanStep(
          index: 0,
          description: 'Step 1',
          status: PlanStepStatus.completed,
        ),
        PlanStep(
          index: 1,
          description: 'Step 2',
          status: PlanStepStatus.completed,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [activePlanProvider('session-1').overrideWithValue(plan)],
        child: const MaterialApp(home: PlanViewScreen(sessionId: 'session-1')),
      ),
    );

    await tester.pump();

    expect(find.text('100% complete  \u00b7  2/2 steps'), findsOneWidget);
  });
}

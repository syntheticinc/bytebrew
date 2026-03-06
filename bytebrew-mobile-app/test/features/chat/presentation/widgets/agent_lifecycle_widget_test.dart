import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/agent_lifecycle_widget.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

final _now = DateTime.now();

ChatMessage _makeLifecycleMessage(String content) {
  return ChatMessage(
    id: 'msg-lifecycle',
    type: ChatMessageType.systemMessage,
    content: content,
    timestamp: _now,
  );
}

Widget _buildWidget(ChatMessage message) {
  return MaterialApp(
    home: Scaffold(body: AgentLifecycleWidget(message: message)),
  );
}

void main() {
  testWidgets('AgentLifecycleWidget shows "Agent started" for spawn events', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent started: CodeReviewer');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    expect(find.text('Agent started'), findsOneWidget);
    expect(find.text('CodeReviewer'), findsOneWidget);
  });

  testWidgets(
    'AgentLifecycleWidget shows "Agent completed" for completed events',
    (tester) async {
      final message = _makeLifecycleMessage('Agent completed: TestRunner');

      await tester.pumpWidget(_buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('Agent completed'), findsOneWidget);
      expect(find.text('TestRunner'), findsOneWidget);
    },
  );

  testWidgets('AgentLifecycleWidget shows "Agent failed" for failure events', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent failed: BuildAgent');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    expect(find.text('Agent failed'), findsOneWidget);
    expect(find.text('BuildAgent'), findsOneWidget);
  });

  testWidgets('AgentLifecycleWidget shows play icon for started events', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent started: Worker');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.play_circle_outline), findsOneWidget);
  });

  testWidgets('AgentLifecycleWidget shows check icon for completed events', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent completed: Worker');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
  });

  testWidgets('AgentLifecycleWidget shows cancel icon for failed events', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent failed: Worker');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.cancel_outlined), findsOneWidget);
  });

  testWidgets(
    'AgentLifecycleWidget shows expand icon when description is present',
    (tester) async {
      final message = _makeLifecycleMessage(
        'Agent started: CodeReviewer\nReviewing pull request #42',
      );

      await tester.pumpWidget(_buildWidget(message));
      await tester.pumpAndSettle();

      // Expand icon should be visible (initially collapsed).
      expect(find.byIcon(Icons.expand_more), findsOneWidget);
      expect(find.byIcon(Icons.expand_less), findsNothing);
    },
  );

  testWidgets('AgentLifecycleWidget hides expand icon when no description', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent started: SimpleAgent');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    // No expand icon when there's no multi-line description.
    expect(find.byIcon(Icons.expand_more), findsNothing);
    expect(find.byIcon(Icons.expand_less), findsNothing);
  });

  testWidgets('Tapping expands to show description', (tester) async {
    final message = _makeLifecycleMessage(
      'Agent started: Reviewer\nAnalyzing code quality and test coverage',
    );

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    // Description should not be visible initially.
    expect(find.text('Analyzing code quality and test coverage'), findsNothing);

    // Tap to expand.
    await tester.tap(find.byType(InkWell));
    await tester.pumpAndSettle();

    // Description should now be visible.
    expect(
      find.text('Analyzing code quality and test coverage'),
      findsOneWidget,
    );
    expect(find.byIcon(Icons.expand_less), findsOneWidget);
  });

  testWidgets('Tapping again collapses description', (tester) async {
    final message = _makeLifecycleMessage(
      'Agent started: Worker\nDoing some work',
    );

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    // Expand.
    await tester.tap(find.byType(InkWell));
    await tester.pumpAndSettle();
    expect(find.text('Doing some work'), findsOneWidget);

    // Collapse.
    await tester.tap(find.byType(InkWell));
    await tester.pumpAndSettle();
    expect(find.text('Doing some work'), findsNothing);
  });

  testWidgets('AgentLifecycleWidget handles content without colon', (
    tester,
  ) async {
    final message = _makeLifecycleMessage('Agent started');

    await tester.pumpWidget(_buildWidget(message));
    await tester.pumpAndSettle();

    // Should still render "Agent started" header without crash.
    expect(find.text('Agent started'), findsOneWidget);
  });
}

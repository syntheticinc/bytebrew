import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/reasoning_widget.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ReasoningWidget', () {
    Widget buildWidget(ChatMessage message) {
      return MaterialApp(
        home: Scaffold(body: ReasoningWidget(message: message)),
      );
    }

    ChatMessage buildReasoningMessage(String content) {
      return ChatMessage(
        id: 'reasoning-1',
        type: ChatMessageType.reasoning,
        content: content,
        timestamp: DateTime.now(),
      );
    }

    testWidgets('renders "Thinking..." label', (tester) async {
      final message = buildReasoningMessage('Some reasoning text');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('Thinking...'), findsOneWidget);
    });

    testWidgets('"Thinking..." text has italic style', (tester) async {
      final message = buildReasoningMessage('Analyzing the code...');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final thinkingText = tester.widget<Text>(find.text('Thinking...'));
      expect(thinkingText.style?.fontStyle, FontStyle.italic);
    });

    testWidgets('"Thinking..." text uses shade3 color', (tester) async {
      final message = buildReasoningMessage('Processing...');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final thinkingText = tester.widget<Text>(find.text('Thinking...'));
      expect(thinkingText.style?.color, AppColors.shade3);
    });

    testWidgets('has left border decoration', (tester) async {
      final message = buildReasoningMessage('Reasoning content');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // The inner Container has a left border.
      final containers = tester.widgetList<Container>(find.byType(Container));
      final borderedContainer = containers.where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.border != null) {
          final border = decoration.border;
          if (border is Border) {
            return border.left.width == 2 &&
                border.left.color == AppColors.shade3;
          }
        }
        return false;
      });
      expect(borderedContainer, isNotEmpty);
    });

    testWidgets('tap opens bottom sheet with reasoning content', (
      tester,
    ) async {
      final message = buildReasoningMessage(
        'The user wants to refactor the auth module.',
      );
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // Tap on the "Thinking..." label.
      await tester.tap(find.text('Thinking...'));
      await tester.pumpAndSettle();

      // Bottom sheet should appear with "Agent Reasoning" header.
      expect(find.text('Agent Reasoning'), findsOneWidget);

      // The reasoning content should be visible inside the bottom sheet.
      expect(
        find.text('The user wants to refactor the auth module.'),
        findsOneWidget,
      );
    });

    testWidgets('bottom sheet has auto_awesome icon', (tester) async {
      final message = buildReasoningMessage('Some reasoning');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Thinking...'));
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.auto_awesome), findsOneWidget);
    });

    testWidgets('bottom sheet reasoning text is italic', (tester) async {
      final message = buildReasoningMessage('Italic reasoning text');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Thinking...'));
      await tester.pumpAndSettle();

      final reasoningText = tester.widget<Text>(
        find.text('Italic reasoning text'),
      );
      expect(reasoningText.style?.fontStyle, FontStyle.italic);
    });

    testWidgets('does not show raw reasoning content in main view', (
      tester,
    ) async {
      final message = buildReasoningMessage(
        'Hidden reasoning content that should only show in sheet',
      );
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // The raw reasoning content should NOT be visible in the main widget.
      expect(
        find.text('Hidden reasoning content that should only show in sheet'),
        findsNothing,
      );

      // Only "Thinking..." should be visible.
      expect(find.text('Thinking...'), findsOneWidget);
    });

    testWidgets('is wrapped in InkWell for tap handling', (tester) async {
      final message = buildReasoningMessage('Tappable');
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.byType(InkWell), findsOneWidget);
    });
  });
}

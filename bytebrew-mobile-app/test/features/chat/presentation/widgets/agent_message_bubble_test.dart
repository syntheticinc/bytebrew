import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/agent_message_bubble.dart';
import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AgentMessageBubble', () {
    Widget buildWidget(ChatMessage message) {
      return MaterialApp(
        home: Scaffold(body: AgentMessageBubble(message: message)),
      );
    }

    testWidgets('renders plain text content via MarkdownBody', (tester) async {
      final message = ChatMessage(
        id: 'msg-1',
        type: ChatMessageType.agentMessage,
        content: 'Hello, how can I help?',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // MarkdownBody should be present.
      expect(find.byType(MarkdownBody), findsOneWidget);

      // The text content should be rendered.
      expect(find.text('Hello, how can I help?'), findsOneWidget);
    });

    testWidgets('renders markdown with bold text', (tester) async {
      final message = ChatMessage(
        id: 'msg-2',
        type: ChatMessageType.agentMessage,
        content: 'This is **bold** text',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.byType(MarkdownBody), findsOneWidget);
      // The word "bold" should appear in the rendered output.
      expect(find.textContaining('bold'), findsOneWidget);
    });

    testWidgets('renders markdown with bullet list', (tester) async {
      final message = ChatMessage(
        id: 'msg-3',
        type: ChatMessageType.agentMessage,
        content: '- Item one\n- Item two\n- Item three',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.byType(MarkdownBody), findsOneWidget);
      expect(find.textContaining('Item one'), findsOneWidget);
      expect(find.textContaining('Item two'), findsOneWidget);
      expect(find.textContaining('Item three'), findsOneWidget);
    });

    testWidgets('has accent-colored left indicator bar', (tester) async {
      final message = ChatMessage(
        id: 'msg-4',
        type: ChatMessageType.agentMessage,
        content: 'With indicator',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // Find the small accent bar (3px wide, 16px high).
      final containers = tester.widgetList<Container>(
        find.byType(Container),
      );
      final accentBar = containers.where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && c.constraints != null) {
          return c.constraints!.maxWidth == 3 &&
              decoration.color == AppColors.accent;
        }
        return false;
      });
      expect(accentBar, isNotEmpty);
    });

    testWidgets('uses Row layout with indicator and expanded markdown', (
      tester,
    ) async {
      final message = ChatMessage(
        id: 'msg-5',
        type: ChatMessageType.agentMessage,
        content: 'Layout test',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // The widget uses a Row at the top level of its Padding child.
      expect(find.byType(Row), findsWidgets);
      expect(find.byType(MarkdownBody), findsOneWidget);
    });

    testWidgets('renders in dark theme without errors', (tester) async {
      final message = ChatMessage(
        id: 'msg-6',
        type: ChatMessageType.agentMessage,
        content: 'Dark mode content',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData.dark(),
          home: Scaffold(body: AgentMessageBubble(message: message)),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Dark mode content'), findsOneWidget);
    });
  });
}

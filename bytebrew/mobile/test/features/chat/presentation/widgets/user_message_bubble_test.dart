import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/user_message_bubble.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UserMessageBubble', () {
    Widget buildWidget(ChatMessage message) {
      return MaterialApp(
        home: Scaffold(body: UserMessageBubble(message: message)),
      );
    }

    testWidgets('renders message text', (tester) async {
      final message = ChatMessage(
        id: 'user-msg-1',
        type: ChatMessageType.userMessage,
        content: 'Hello agent, help me with tests',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('Hello agent, help me with tests'), findsOneWidget);
    });

    testWidgets('is aligned to the right', (tester) async {
      final message = ChatMessage(
        id: 'user-msg-2',
        type: ChatMessageType.userMessage,
        content: 'Right aligned?',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final align = tester.widget<Align>(find.byType(Align));
      expect(align.alignment, Alignment.centerRight);
    });

    testWidgets('has accent background color', (tester) async {
      final message = ChatMessage(
        id: 'user-msg-3',
        type: ChatMessageType.userMessage,
        content: 'Styled message',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // Find the container with the accent background.
      final containers = tester.widgetList<Container>(find.byType(Container));
      final accentContainer = containers.where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration) {
          return decoration.color == AppColors.accent;
        }
        return false;
      });
      expect(accentContainer, isNotEmpty);
    });

    testWidgets('text uses light color for contrast on accent background', (
      tester,
    ) async {
      final message = ChatMessage(
        id: 'user-msg-4',
        type: ChatMessageType.userMessage,
        content: 'Light text',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final text = tester.widget<Text>(find.text('Light text'));
      expect(text.style?.color, AppColors.light);
    });

    testWidgets('has asymmetric border radius (chat bubble shape)', (
      tester,
    ) async {
      final message = ChatMessage(
        id: 'user-msg-5',
        type: ChatMessageType.userMessage,
        content: 'Bubble shape',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final containers = tester.widgetList<Container>(find.byType(Container));
      final bubbleContainer = containers.firstWhere((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration &&
            decoration.color == AppColors.accent) {
          return decoration.borderRadius != null;
        }
        return false;
      });

      final decoration = bubbleContainer.decoration as BoxDecoration;
      final borderRadius = decoration.borderRadius as BorderRadius;

      // Top-left, top-right, bottom-left are 12, bottom-right is 2.
      expect(borderRadius.topLeft, const Radius.circular(12));
      expect(borderRadius.topRight, const Radius.circular(12));
      expect(borderRadius.bottomLeft, const Radius.circular(12));
      expect(borderRadius.bottomRight, const Radius.circular(2));
    });

    testWidgets('renders long text without overflow', (tester) async {
      final message = ChatMessage(
        id: 'user-msg-6',
        type: ChatMessageType.userMessage,
        content: 'A' * 500,
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // Should render without exceptions.
      expect(find.byType(UserMessageBubble), findsOneWidget);
    });

    testWidgets('renders correctly in dark theme', (tester) async {
      final message = ChatMessage(
        id: 'user-msg-7',
        type: ChatMessageType.userMessage,
        content: 'Dark mode message',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData.dark(),
          home: Scaffold(body: UserMessageBubble(message: message)),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Dark mode message'), findsOneWidget);
    });
  });
}

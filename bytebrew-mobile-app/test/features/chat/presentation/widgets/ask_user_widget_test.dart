import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/ask_user_widget.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../../helpers/fakes.dart';

const _sessionId = 'test-session';

void main() {
  group('AskUserWidget', () {
    Widget buildWidget(ChatMessage message) {
      return ProviderScope(
        overrides: [
          chatRepositoryProvider.overrideWithValue(FakeChatRepository([])),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: AskUserWidget(message: message, sessionId: _sessionId),
          ),
        ),
      );
    }

    testWidgets('renders SizedBox.shrink when askUser is null', (
      tester,
    ) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: null,
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final sizedBox = tester.widget<SizedBox>(find.byType(SizedBox).first);
      expect(sizedBox.width, 0.0);
      expect(sizedBox.height, 0.0);
    });

    testWidgets('renders question text', (tester) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Which auth method to use?',
          options: ['JWT', 'Session cookies'],
          status: AskUserStatus.pending,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('Which auth method to use?'), findsOneWidget);
    });

    testWidgets('renders all options as tappable items when pending', (
      tester,
    ) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Pick a framework',
          options: ['React', 'Vue', 'Svelte'],
          status: AskUserStatus.pending,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('React'), findsOneWidget);
      expect(find.text('Vue'), findsOneWidget);
      expect(find.text('Svelte'), findsOneWidget);
    });

    testWidgets('does not show options when status is answered', (
      tester,
    ) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Pick a framework',
          options: ['React', 'Vue'],
          status: AskUserStatus.answered,
          answer: 'React',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // Options should not appear as tappable tiles.
      // Only the answered badge shows "Answered: React".
      expect(find.text('Answered: React'), findsOneWidget);
    });

    testWidgets('answered state shows check_circle icon', (tester) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Pick a color',
          options: ['Red', 'Blue'],
          status: AskUserStatus.answered,
          answer: 'Blue',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.check_circle), findsOneWidget);

      final icon = tester.widget<Icon>(find.byIcon(Icons.check_circle));
      expect(icon.color, AppColors.statusActive);
    });

    testWidgets('has accent border on the container', (tester) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Question?',
          options: ['A'],
          status: AskUserStatus.pending,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // The outer container has an accent border.
      final containers = tester.widgetList<Container>(
        find.byType(Container),
      );
      final outerContainer = containers.firstWhere(
        (c) {
          final decoration = c.decoration;
          if (decoration is BoxDecoration && decoration.border != null) {
            final border = decoration.border;
            if (border is Border) {
              return border.top.color == AppColors.accent;
            }
          }
          return false;
        },
        orElse: () => Container(),
      );
      expect(outerContainer.decoration, isNotNull);
    });

    testWidgets('question text has bold font weight', (tester) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Important question',
          options: ['Yes'],
          status: AskUserStatus.pending,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final questionText = tester.widget<Text>(
        find.text('Important question'),
      );
      expect(questionText.style?.fontWeight, FontWeight.bold);
    });

    testWidgets('answered badge shows italic text', (tester) async {
      final message = ChatMessage(
        id: 'ask-msg-1',
        type: ChatMessageType.askUser,
        content: '',
        timestamp: DateTime.now(),
        askUser: const AskUserData(
          id: 'ask-1',
          question: 'Q?',
          options: ['A', 'B'],
          status: AskUserStatus.answered,
          answer: 'A',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final answeredText = tester.widget<Text>(find.text('Answered: A'));
      expect(answeredText.style?.fontStyle, FontStyle.italic);
    });
  });
}

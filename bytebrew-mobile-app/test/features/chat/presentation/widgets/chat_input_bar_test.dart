import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/chat_input_bar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../../helpers/fakes.dart';

const _sessionId = 'test-session';

void main() {
  group('ChatInputBar', () {
    late StreamableFakeChatRepository fakeRepo;

    setUp(() {
      fakeRepo = StreamableFakeChatRepository();
    });

    tearDown(() {
      fakeRepo.dispose();
    });

    Widget buildWidget({bool enabled = true}) {
      return ProviderScope(
        overrides: [
          chatRepositoryProvider.overrideWithValue(fakeRepo),
          sessionChatRepositoryProvider(_sessionId)
              .overrideWithValue(fakeRepo),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: ChatInputBar(sessionId: _sessionId, enabled: enabled),
          ),
        ),
      );
    }

    testWidgets('renders text field', (tester) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      expect(find.byType(TextField), findsOneWidget);
    });

    testWidgets('renders send button (IconButton)', (tester) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.arrow_upward), findsOneWidget);
    });

    testWidgets('shows hint text when enabled', (tester) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      expect(find.text('Ask your agent...'), findsOneWidget);
    });

    testWidgets('shows different hint text when disabled', (tester) async {
      await tester.pumpWidget(buildWidget(enabled: false));
      await tester.pumpAndSettle();

      expect(
        find.text('Switch to Supervisor to send messages'),
        findsOneWidget,
      );
    });

    testWidgets('send button is disabled when text is empty', (tester) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      // Find the IconButton and verify it has null onPressed.
      final iconButton = tester.widget<IconButton>(
        find.ancestor(
          of: find.byIcon(Icons.arrow_upward),
          matching: find.byType(IconButton),
        ),
      );
      expect(iconButton.onPressed, isNull);
    });

    testWidgets('send button becomes enabled after entering text', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      // Enter text.
      await tester.enterText(find.byType(TextField), 'Hello agent');
      await tester.pump();

      // Now the IconButton should have a non-null onPressed.
      final iconButton = tester.widget<IconButton>(
        find.ancestor(
          of: find.byIcon(Icons.arrow_upward),
          matching: find.byType(IconButton),
        ),
      );
      expect(iconButton.onPressed, isNotNull);
    });

    testWidgets('send button stays disabled with only whitespace', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField), '   ');
      await tester.pump();

      final iconButton = tester.widget<IconButton>(
        find.ancestor(
          of: find.byIcon(Icons.arrow_upward),
          matching: find.byType(IconButton),
        ),
      );
      expect(iconButton.onPressed, isNull);
    });

    testWidgets('tapping send calls sendMessage and clears text field', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      // Enter text.
      await tester.enterText(find.byType(TextField), 'Test message');
      await tester.pump();

      // Tap the send button.
      await tester.tap(
        find.ancestor(
          of: find.byIcon(Icons.arrow_upward),
          matching: find.byType(IconButton),
        ),
      );
      await tester.pump();

      // Verify the message was sent through the repository.
      expect(fakeRepo.sentMessages, contains('Test message'));

      // Verify the text field was cleared.
      final textField = tester.widget<TextField>(find.byType(TextField));
      expect(textField.controller?.text, isEmpty);
    });

    testWidgets('submitting via keyboard sends message', (tester) async {
      await tester.pumpWidget(buildWidget());
      await tester.pumpAndSettle();

      // Enter text and submit via keyboard action.
      await tester.enterText(find.byType(TextField), 'Keyboard submit');
      await tester.testTextInput.receiveAction(TextInputAction.send);
      await tester.pump();

      expect(fakeRepo.sentMessages, contains('Keyboard submit'));
    });

    testWidgets('text field is disabled when enabled=false', (tester) async {
      await tester.pumpWidget(buildWidget(enabled: false));
      await tester.pumpAndSettle();

      final textField = tester.widget<TextField>(find.byType(TextField));
      expect(textField.enabled, isFalse);
    });

    testWidgets('send button is disabled when enabled=false even with text', (
      tester,
    ) async {
      await tester.pumpWidget(buildWidget(enabled: false));
      await tester.pumpAndSettle();

      // Cannot enter text because TextField is disabled, but verify the
      // IconButton is also disabled.
      final iconButton = tester.widget<IconButton>(
        find.ancestor(
          of: find.byIcon(Icons.arrow_upward),
          matching: find.byType(IconButton),
        ),
      );
      expect(iconButton.onPressed, isNull);
    });
  });
}

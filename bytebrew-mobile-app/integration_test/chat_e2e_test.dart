import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/presentation/chat_screen.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/sessions_screen.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../test/helpers/fakes.dart';
import 'helpers/test_app.dart';

final _testSession = Session(
  id: 'e2e-session',
  serverId: 'srv-1',
  serverName: 'Test Server',
  projectName: 'e2e-project',
  status: SessionStatus.active,
  currentTask: 'Running E2E tests',
  hasAskUser: false,
  lastActivityAt: DateTime.now(),
);

void main() {
  initializeBinding();

  group('TC-E2E-CHAT: Full chat flow', () {
    testWidgets(
      'TC-E2E-CHAT-01: Authenticated -> sessions -> tap session -> chat',
      (tester) async {
        final chatRepo = StreamableFakeChatRepository();

        addTearDown(chatRepo.dispose);

        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authProvider.overrideWithValue(const AuthState.authenticated()),
              authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
              sessionsProvider
                  .overrideWith(() => FakeSessionsNotifier([_testSession])),
              groupedSessionsProvider.overrideWithValue({
                SessionStatus.active: [_testSession],
              }),
              chatRepositoryProvider.overrideWithValue(chatRepo),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Wait for splash -> sessions redirect (authenticated).
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        // Should be on SessionsScreen.
        expect(find.byType(SessionsScreen), findsOneWidget);

        // Tap on the session card to navigate to chat.
        // Session cards show the project name.
        await tester.tap(find.text('e2e-project'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        // Should navigate to ChatScreen.
        expect(find.byType(ChatScreen), findsOneWidget);

        // Initially empty — shows placeholder.
        expect(find.text('Start a conversation'), findsOneWidget);
      },
    );

    testWidgets(
      'TC-E2E-CHAT-02: Send message and receive streamed response',
      (tester) async {
        final chatRepo = StreamableFakeChatRepository();

        addTearDown(chatRepo.dispose);

        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authProvider.overrideWithValue(const AuthState.authenticated()),
              authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
              sessionsProvider
                  .overrideWith(() => FakeSessionsNotifier([_testSession])),
              groupedSessionsProvider.overrideWithValue({
                SessionStatus.active: [_testSession],
              }),
              chatRepositoryProvider.overrideWithValue(chatRepo),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Navigate past splash to sessions.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        // Tap session card to go to chat.
        await tester.tap(find.text('e2e-project'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        expect(find.byType(ChatScreen), findsOneWidget);

        // Find the chat input and enter a message.
        final inputFinder = find.byType(TextField);
        await tester.enterText(inputFinder, 'Hello agent');
        await tester.pump();

        // Submit via text input action.
        await tester.testTextInput.receiveAction(TextInputAction.send);
        await tester.pump();

        // Verify the message was sent through the repository.
        expect(chatRepo.sentMessages, contains('Hello agent'));

        // Simulate server response via stream.
        chatRepo.emitMessages([
          ChatMessage(
            id: 'user-1',
            type: ChatMessageType.userMessage,
            content: 'Hello agent',
            timestamp: DateTime.now().subtract(const Duration(seconds: 5)),
          ),
          ChatMessage(
            id: 'agent-1',
            type: ChatMessageType.agentMessage,
            content: 'Hello! How can I help you today?',
            timestamp: DateTime.now(),
          ),
        ]);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // Both messages should be visible.
        expect(find.text('Hello agent'), findsOneWidget);
        expect(find.text('Hello! How can I help you today?'), findsOneWidget);
      },
    );

    testWidgets(
      'TC-E2E-CHAT-03: Empty input does not send a message',
      (tester) async {
        final chatRepo = StreamableFakeChatRepository();

        addTearDown(chatRepo.dispose);

        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authProvider.overrideWithValue(const AuthState.authenticated()),
              authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
              sessionsProvider
                  .overrideWith(() => FakeSessionsNotifier([_testSession])),
              groupedSessionsProvider.overrideWithValue({
                SessionStatus.active: [_testSession],
              }),
              chatRepositoryProvider.overrideWithValue(chatRepo),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Navigate to chat.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));
        await tester.tap(find.text('e2e-project'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        // Try to submit empty input.
        await tester.testTextInput.receiveAction(TextInputAction.send);
        await tester.pump();

        // No message should have been sent.
        expect(chatRepo.sentMessages, isEmpty);
      },
    );
  });
}

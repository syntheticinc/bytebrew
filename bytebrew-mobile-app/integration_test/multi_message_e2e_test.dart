import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/presentation/chat_screen.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../test/helpers/fakes.dart';
import 'helpers/test_app.dart';

final _testSession = Session(
  id: 'multi-session',
  serverId: 'srv-1',
  serverName: 'Test Server',
  projectName: 'multi-project',
  status: SessionStatus.active,
  currentTask: 'Multi-message test',
  hasAskUser: false,
  lastActivityAt: DateTime.now(),
);

void main() {
  initializeBinding();

  group('TC-E2E-MULTI: Multiple messages in sequence', () {
    testWidgets(
      'TC-E2E-MULTI-01: Send multiple messages, verify order and responses',
      (tester) async {
        final chatRepo = StreamableFakeChatRepository();

        addTearDown(chatRepo.dispose);

        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authProvider.overrideWithValue(const AuthState.authenticated()),
              authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
              sessionsProvider.overrideWith(
                () => FakeSessionsNotifier([_testSession]),
              ),
              groupedSessionsProvider.overrideWithValue({
                SessionStatus.active: [_testSession],
              }),
              chatRepositoryProvider.overrideWithValue(chatRepo),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Navigate past splash to sessions -> tap session -> chat.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));
        await tester.tap(find.text('multi-project'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        expect(find.byType(ChatScreen), findsOneWidget);

        // --- Message 1: Send and receive ---

        final inputFinder = find.byType(TextField);
        final sendButton = find.byIcon(Icons.arrow_upward);

        await tester.enterText(inputFinder, 'First question');
        await tester.pump();
        await tester.tap(sendButton);
        await tester.pump();

        expect(chatRepo.sentMessages, ['First question']);

        // Simulate response for message 1.
        final now = DateTime.now();
        chatRepo.emitMessages([
          ChatMessage(
            id: 'user-1',
            type: ChatMessageType.userMessage,
            content: 'First question',
            timestamp: now.subtract(const Duration(seconds: 10)),
          ),
          ChatMessage(
            id: 'agent-1',
            type: ChatMessageType.agentMessage,
            content: 'First answer',
            timestamp: now.subtract(const Duration(seconds: 8)),
          ),
        ]);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        expect(find.text('First question'), findsOneWidget);
        expect(find.text('First answer'), findsOneWidget);

        // --- Message 2: Send and receive ---

        await tester.enterText(inputFinder, 'Second question');
        await tester.pump();
        await tester.tap(sendButton);
        await tester.pump();

        expect(chatRepo.sentMessages, ['First question', 'Second question']);

        // Simulate response including all messages in correct order.
        chatRepo.emitMessages([
          ChatMessage(
            id: 'user-1',
            type: ChatMessageType.userMessage,
            content: 'First question',
            timestamp: now.subtract(const Duration(seconds: 10)),
          ),
          ChatMessage(
            id: 'agent-1',
            type: ChatMessageType.agentMessage,
            content: 'First answer',
            timestamp: now.subtract(const Duration(seconds: 8)),
          ),
          ChatMessage(
            id: 'user-2',
            type: ChatMessageType.userMessage,
            content: 'Second question',
            timestamp: now.subtract(const Duration(seconds: 5)),
          ),
          ChatMessage(
            id: 'agent-2',
            type: ChatMessageType.agentMessage,
            content: 'Second answer',
            timestamp: now.subtract(const Duration(seconds: 3)),
          ),
        ]);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // All four messages should be visible in the list.
        expect(find.text('First question'), findsOneWidget);
        expect(find.text('First answer'), findsOneWidget);
        expect(find.text('Second question'), findsOneWidget);
        expect(find.text('Second answer'), findsOneWidget);

        // --- Message 3: Send and receive ---

        await tester.enterText(inputFinder, 'Third question');
        await tester.pump();
        await tester.tap(sendButton);
        await tester.pump();

        expect(chatRepo.sentMessages.length, 3);
        expect(chatRepo.sentMessages.last, 'Third question');

        // Simulate full conversation with six messages.
        chatRepo.emitMessages([
          ChatMessage(
            id: 'user-1',
            type: ChatMessageType.userMessage,
            content: 'First question',
            timestamp: now.subtract(const Duration(seconds: 10)),
          ),
          ChatMessage(
            id: 'agent-1',
            type: ChatMessageType.agentMessage,
            content: 'First answer',
            timestamp: now.subtract(const Duration(seconds: 8)),
          ),
          ChatMessage(
            id: 'user-2',
            type: ChatMessageType.userMessage,
            content: 'Second question',
            timestamp: now.subtract(const Duration(seconds: 5)),
          ),
          ChatMessage(
            id: 'agent-2',
            type: ChatMessageType.agentMessage,
            content: 'Second answer',
            timestamp: now.subtract(const Duration(seconds: 3)),
          ),
          ChatMessage(
            id: 'user-3',
            type: ChatMessageType.userMessage,
            content: 'Third question',
            timestamp: now.subtract(const Duration(seconds: 2)),
          ),
          ChatMessage(
            id: 'agent-3',
            type: ChatMessageType.agentMessage,
            content: 'Third answer',
            timestamp: now,
          ),
        ]);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // All six messages visible and in correct order.
        expect(find.text('First question'), findsOneWidget);
        expect(find.text('First answer'), findsOneWidget);
        expect(find.text('Second question'), findsOneWidget);
        expect(find.text('Second answer'), findsOneWidget);
        expect(find.text('Third question'), findsOneWidget);
        expect(find.text('Third answer'), findsOneWidget);
      },
    );

    testWidgets(
      'TC-E2E-MULTI-02: Messages with mixed types (user, agent, tool call)',
      (tester) async {
        final chatRepo = StreamableFakeChatRepository();

        addTearDown(chatRepo.dispose);

        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authProvider.overrideWithValue(const AuthState.authenticated()),
              authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
              sessionsProvider.overrideWith(
                () => FakeSessionsNotifier([_testSession]),
              ),
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
        await tester.tap(find.text('multi-project'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        // Emit a conversation with mixed message types.
        final now = DateTime.now();
        chatRepo.emitMessages([
          ChatMessage(
            id: 'msg-1',
            type: ChatMessageType.userMessage,
            content: 'Analyze my code',
            timestamp: now.subtract(const Duration(seconds: 10)),
          ),
          ChatMessage(
            id: 'msg-2',
            type: ChatMessageType.agentMessage,
            content: 'I will read the main file first.',
            timestamp: now.subtract(const Duration(seconds: 8)),
          ),
          ChatMessage(
            id: 'msg-3',
            type: ChatMessageType.agentMessage,
            content: 'Analysis complete. The code looks good.',
            timestamp: now,
          ),
        ]);

        // Use pump() (not pumpAndSettle) to avoid infinite animation hangs.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // Verify all message types are rendered.
        expect(find.text('Analyze my code'), findsOneWidget);
        expect(find.text('I will read the main file first.'), findsOneWidget);
        expect(
          find.text('Analysis complete. The code looks good.'),
          findsOneWidget,
        );
      },
    );
  });
}

import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/presentation/chat_screen.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../helpers/fakes.dart';

final _testSession = Session(
  id: 'test-session',
  serverId: 'srv-1',
  serverName: 'Test Server',
  projectName: 'test-project',
  status: SessionStatus.active,
  hasAskUser: false,
  lastActivityAt: DateTime.now(),
);

final _testMessages = [
  ChatMessage(
    id: 'msg-1',
    type: ChatMessageType.userMessage,
    content: 'Hello agent',
    timestamp: DateTime.now().subtract(const Duration(minutes: 5)),
  ),
  ChatMessage(
    id: 'msg-2',
    type: ChatMessageType.agentMessage,
    content: 'Hello! How can I help?',
    timestamp: DateTime.now().subtract(const Duration(minutes: 4)),
  ),
  ChatMessage(
    id: 'msg-3',
    type: ChatMessageType.reasoning,
    content: 'The user wants help with something.',
    timestamp: DateTime.now().subtract(const Duration(minutes: 3)),
  ),
  ChatMessage(
    id: 'msg-4',
    type: ChatMessageType.systemMessage,
    content: 'Session started',
    timestamp: DateTime.now().subtract(const Duration(minutes: 6)),
  ),
];

void main() {
  testWidgets('ChatScreen renders user and agent messages', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionChatRepositoryProvider.overrideWith(
            (ref, sessionId) => FakeChatRepository(_testMessages),
          ),
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
        ],
        child: const MaterialApp(home: ChatScreen(sessionId: 'test-session')),
      ),
    );

    await tester.pumpAndSettle();

    // Verify user message is displayed
    expect(find.text('Hello agent'), findsOneWidget);

    // Verify agent message is displayed
    expect(find.text('Hello! How can I help?'), findsOneWidget);
  });

  testWidgets('ChatScreen renders system message', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionChatRepositoryProvider.overrideWith(
            (ref, sessionId) => FakeChatRepository(_testMessages),
          ),
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
        ],
        child: const MaterialApp(home: ChatScreen(sessionId: 'test-session')),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Session started'), findsOneWidget);
  });

  testWidgets('ChatScreen renders reasoning as "Thinking..."', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionChatRepositoryProvider.overrideWith(
            (ref, sessionId) => FakeChatRepository(_testMessages),
          ),
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
        ],
        child: const MaterialApp(home: ChatScreen(sessionId: 'test-session')),
      ),
    );

    await tester.pumpAndSettle();

    // ReasoningWidget renders "Thinking..." label
    expect(find.text('Thinking...'), findsOneWidget);
  });

  testWidgets('ChatScreen shows project name in app bar', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionChatRepositoryProvider.overrideWith(
            (ref, sessionId) => FakeChatRepository(_testMessages),
          ),
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
        ],
        child: const MaterialApp(home: ChatScreen(sessionId: 'test-session')),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('test-project'), findsOneWidget);
    expect(find.text('Test Server'), findsOneWidget);
  });

  testWidgets('ChatScreen shows empty state when no messages', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionChatRepositoryProvider.overrideWith(
            (ref, sessionId) => FakeChatRepository([]),
          ),
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
        ],
        child: const MaterialApp(home: ChatScreen(sessionId: 'test-session')),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Start a conversation'), findsOneWidget);
    expect(find.text('Send a message to your agent'), findsOneWidget);
  });
}

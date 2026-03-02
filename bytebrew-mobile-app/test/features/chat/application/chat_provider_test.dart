import 'dart:async';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/empty_chat_repository.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../helpers/fakes.dart';

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

final _now = DateTime.now();

final _testSession = Session(
  id: 'session-1',
  serverId: 'srv-1',
  serverName: 'Test Server',
  projectName: 'test-project',
  status: SessionStatus.active,
  hasAskUser: false,
  lastActivityAt: _now,
);

final _testMessages = [
  ChatMessage(
    id: 'msg-1',
    type: ChatMessageType.userMessage,
    content: 'Hello',
    timestamp: _now.subtract(const Duration(minutes: 5)),
  ),
  ChatMessage(
    id: 'msg-2',
    type: ChatMessageType.agentMessage,
    content: 'Hi there!',
    timestamp: _now.subtract(const Duration(minutes: 4)),
  ),
];

final _planMessages = [
  ChatMessage(
    id: 'msg-p1',
    type: ChatMessageType.planUpdate,
    content: 'Plan created',
    timestamp: _now.subtract(const Duration(minutes: 3)),
    plan: const PlanData(
      goal: 'Refactor auth',
      steps: [
        PlanStep(
          index: 0,
          description: 'Analyze code',
          status: PlanStepStatus.completed,
        ),
        PlanStep(
          index: 1,
          description: 'Write tests',
          status: PlanStepStatus.inProgress,
        ),
      ],
    ),
  ),
];

final _askUserMessages = [
  ChatMessage(
    id: 'msg-a1',
    type: ChatMessageType.askUser,
    content: 'Which approach?',
    timestamp: _now.subtract(const Duration(minutes: 2)),
    askUser: const AskUserData(
      id: 'ask-1',
      question: 'Which approach?',
      options: ['Option A', 'Option B'],
      status: AskUserStatus.pending,
    ),
  ),
];

void main() {
  // =========================================================================
  // chatRepositoryProvider
  // =========================================================================
  group('chatRepositoryProvider', () {
    test('returns EmptyChatRepository when WS is disconnected', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final repo = container.read(chatRepositoryProvider);
      expect(repo, isA<EmptyChatRepository>());
    });
  });

  // =========================================================================
  // ChatMessages
  // =========================================================================
  group('ChatMessages', () {
    test('builds with messages from repository', () async {
      final fakeRepo = FakeChatRepository(_testMessages);

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          // Override session-specific chat repository to return our fake.
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      // Trigger the async build.
      final future = container.read(chatMessagesProvider('session-1').future);
      final messages = await future;

      expect(messages, hasLength(2));
      expect(messages.first.content, 'Hello');
      expect(messages.last.content, 'Hi there!');
    });

    test('returns empty list when repository has no messages', () async {
      final fakeRepo = FakeChatRepository([]);

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      final messages =
          await container.read(chatMessagesProvider('session-1').future);
      expect(messages, isEmpty);
    });

    test('sendMessage calls repository and re-fetches', () async {
      final fakeRepo = StreamableFakeChatRepository(
        initialMessages: _testMessages,
      );
      addTearDown(fakeRepo.dispose);

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      // Wait for initial load.
      await container.read(chatMessagesProvider('session-1').future);

      // Send a message.
      await container
          .read(chatMessagesProvider('session-1').notifier)
          .sendMessage('New message');

      expect(fakeRepo.sentMessages, contains('New message'));
    });

    test('stream updates replace state', () async {
      final fakeRepo = StreamableFakeChatRepository(
        initialMessages: _testMessages,
      );
      addTearDown(fakeRepo.dispose);

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      // Wait for initial load.
      await container.read(chatMessagesProvider('session-1').future);

      final updatedMessages = [
        ..._testMessages,
        ChatMessage(
          id: 'msg-3',
          type: ChatMessageType.agentMessage,
          content: 'New stream message',
          timestamp: _now,
        ),
      ];

      // Listen for the stream-triggered state update.
      final completer = Completer<List<ChatMessage>>();
      container.listen(chatMessagesProvider('session-1'), (_, next) {
        final v = next.value;
        if (v != null && v.length == 3 && !completer.isCompleted) {
          completer.complete(v);
        }
      });

      // Emit through the stream.
      fakeRepo.emitMessages(updatedMessages);

      final messages = await completer.future.timeout(
        const Duration(seconds: 2),
      );
      expect(messages, hasLength(3));
      expect(messages.last.content, 'New stream message');
    });
  });

  // =========================================================================
  // activePlanProvider
  // =========================================================================
  group('activePlanProvider', () {
    test('returns null when no plan messages exist', () async {
      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(FakeChatRepository(_testMessages)),
        ],
      );
      addTearDown(container.dispose);

      await container.read(chatMessagesProvider('session-1').future);

      final plan = container.read(activePlanProvider('session-1'));
      expect(plan, isNull);
    });

    test('returns latest plan from planUpdate messages', () async {
      final messagesWithPlan = [..._testMessages, ..._planMessages];

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(FakeChatRepository(messagesWithPlan)),
        ],
      );
      addTearDown(container.dispose);

      await container.read(chatMessagesProvider('session-1').future);

      final plan = container.read(activePlanProvider('session-1'));
      expect(plan, isNotNull);
      expect(plan!.goal, 'Refactor auth');
      expect(plan.steps, hasLength(2));
    });
  });

  // =========================================================================
  // pendingAskUserProvider
  // =========================================================================
  group('pendingAskUserProvider', () {
    test('returns null when no ask-user messages', () async {
      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(FakeChatRepository(_testMessages)),
        ],
      );
      addTearDown(container.dispose);

      await container.read(chatMessagesProvider('session-1').future);

      final askUser = container.read(pendingAskUserProvider('session-1'));
      expect(askUser, isNull);
    });

    test('returns pending ask-user message', () async {
      final messagesWithAskUser = [..._testMessages, ..._askUserMessages];

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(FakeChatRepository(messagesWithAskUser)),
        ],
      );
      addTearDown(container.dispose);

      await container.read(chatMessagesProvider('session-1').future);

      final askUser = container.read(pendingAskUserProvider('session-1'));
      expect(askUser, isNotNull);
      expect(askUser!.askUser?.question, 'Which approach?');
      expect(askUser.askUser?.status, AskUserStatus.pending);
    });

    test('returns null when ask-user is already answered', () async {
      final answeredMessages = [
        ..._testMessages,
        ChatMessage(
          id: 'msg-a2',
          type: ChatMessageType.askUser,
          content: 'Which approach?',
          timestamp: _now,
          askUser: const AskUserData(
            id: 'ask-2',
            question: 'Which approach?',
            options: ['A', 'B'],
            status: AskUserStatus.answered,
            answer: 'A',
          ),
        ),
      ];

      final container = ProviderContainer(
        overrides: [
          sessionsProvider.overrideWith(
            () => FakeSessionsNotifier([_testSession]),
          ),
          sessionChatRepositoryProvider('session-1')
              .overrideWithValue(FakeChatRepository(answeredMessages)),
        ],
      );
      addTearDown(container.dispose);

      await container.read(chatMessagesProvider('session-1').future);

      final askUser = container.read(pendingAskUserProvider('session-1'));
      expect(askUser, isNull);
    });
  });
}

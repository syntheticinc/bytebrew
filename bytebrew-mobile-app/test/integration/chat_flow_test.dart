import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/presentation/chat_screen.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../helpers/fakes.dart';

final _testSession = Session(
  id: 'test-session',
  serverId: 'srv-1',
  serverName: 'Test Server',
  projectName: 'test-project',
  status: SessionStatus.active,
  hasAskUser: false,
  lastActivityAt: DateTime.now(),
);

Widget _buildChatScreen({
  required Object chatRepoOverride,
  List<Session>? sessions,
  String sessionId = 'test-session',
}) {
  return ProviderScope(
    overrides: [
      sessionChatRepositoryProvider.overrideWith(
        (ref, sessionId) => chatRepoOverride as ChatRepository,
      ),
      sessionsProvider.overrideWith(
        () => FakeSessionsNotifier(sessions ?? [_testSession]),
      ),
    ],
    child: MaterialApp(home: ChatScreen(sessionId: sessionId)),
  );
}

void main() {
  group('Chat flow integration', () {
    testWidgets('TC-CHAT-01: Renders messages from repository', (tester) async {
      final messages = [
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
      ];

      await tester.pumpWidget(
        _buildChatScreen(chatRepoOverride: FakeChatRepository(messages)),
      );
      await tester.pumpAndSettle();

      expect(find.text('Hello agent'), findsOneWidget);
      expect(find.text('Hello! How can I help?'), findsOneWidget);
    });

    testWidgets('TC-CHAT-02: Send message records text via repository', (
      tester,
    ) async {
      final repo = StreamableFakeChatRepository();

      await tester.pumpWidget(_buildChatScreen(chatRepoOverride: repo));
      await tester.pumpAndSettle();

      // Find the input field and enter text.
      final inputFinder = find.byType(TextField);
      await tester.enterText(inputFinder, 'Test message');
      await tester.pump();

      // Submit via text input action.
      await tester.testTextInput.receiveAction(TextInputAction.send);
      await tester.pump();

      // Verify the message was sent through the repository.
      expect(repo.sentMessages, contains('Test message'));

      repo.dispose();
    });

    testWidgets('TC-CHAT-03: Real-time update via stream shows new messages', (
      tester,
    ) async {
      final repo = StreamableFakeChatRepository();

      await tester.pumpWidget(_buildChatScreen(chatRepoOverride: repo));
      await tester.pumpAndSettle();

      // Initially empty.
      expect(find.text('Start a conversation'), findsOneWidget);

      // Emit new messages via stream.
      repo.emitMessages([
        ChatMessage(
          id: 'stream-1',
          type: ChatMessageType.userMessage,
          content: 'Streamed question',
          timestamp: DateTime.now(),
        ),
        ChatMessage(
          id: 'stream-2',
          type: ChatMessageType.agentMessage,
          content: 'Streamed answer',
          timestamp: DateTime.now(),
        ),
      ]);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Streamed question'), findsOneWidget);
      expect(find.text('Streamed answer'), findsOneWidget);

      repo.dispose();
    });

    testWidgets(
      'TC-CHAT-04: Tool call lifecycle shows running and completed states',
      (tester) async {
        final repo = StreamableFakeChatRepository();

        await tester.pumpWidget(_buildChatScreen(chatRepoOverride: repo));
        await tester.pumpAndSettle();

        // Emit a running tool call.
        repo.emitMessages([
          ChatMessage(
            id: 'tc-1',
            type: ChatMessageType.toolCall,
            content: '',
            timestamp: DateTime.now(),
            toolCall: const ToolCallData(
              id: 'tool-1',
              toolName: 'read_file',
              arguments: {'path': 'main.go'},
              status: ToolCallStatus.running,
            ),
          ),
        ]);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // Tool name should be visible.
        expect(find.text('read_file'), findsOneWidget);
        // Running state shows a CircularProgressIndicator.
        expect(find.byType(CircularProgressIndicator), findsWidgets);

        // Emit completed tool call.
        repo.emitMessages([
          ChatMessage(
            id: 'tc-1',
            type: ChatMessageType.toolCall,
            content: '',
            timestamp: DateTime.now(),
            toolCall: const ToolCallData(
              id: 'tool-1',
              toolName: 'read_file',
              arguments: {'path': 'main.go'},
              status: ToolCallStatus.completed,
              result: '50 lines',
            ),
          ),
        ]);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // Result should be visible.
        expect(find.text('50 lines'), findsOneWidget);

        repo.dispose();
      },
    );

    testWidgets('TC-CHAT-05: AskUser renders question and options', (
      tester,
    ) async {
      final messages = [
        ChatMessage(
          id: 'ask-1',
          type: ChatMessageType.askUser,
          content: '',
          timestamp: DateTime.now(),
          askUser: const AskUserData(
            id: 'ask-id-1',
            question: 'Which framework?',
            options: ['React', 'Vue', 'Angular'],
            status: AskUserStatus.pending,
          ),
        ),
      ];

      await tester.pumpWidget(
        _buildChatScreen(chatRepoOverride: FakeChatRepository(messages)),
      );
      await tester.pumpAndSettle();

      // Question text should be visible.
      expect(find.text('Which framework?'), findsOneWidget);

      // All options should be rendered.
      expect(find.text('React'), findsOneWidget);
      expect(find.text('Vue'), findsOneWidget);
      expect(find.text('Angular'), findsOneWidget);
    });

    testWidgets('TC-CHAT-06: Empty state shows placeholder', (tester) async {
      await tester.pumpWidget(
        _buildChatScreen(chatRepoOverride: FakeChatRepository([])),
      );
      await tester.pumpAndSettle();

      expect(find.text('Start a conversation'), findsOneWidget);
      expect(find.text('Send a message to your agent'), findsOneWidget);
    });

    testWidgets('TC-CHAT-07: Plan update renders goal and steps', (
      tester,
    ) async {
      final messages = [
        ChatMessage(
          id: 'plan-1',
          type: ChatMessageType.planUpdate,
          content: '',
          timestamp: DateTime.now(),
          plan: PlanData(
            goal: 'Refactor auth module',
            steps: [
              PlanStep(
                index: 0,
                description: 'Extract TokenService',
                status: PlanStepStatus.completed,
                completedAt: DateTime.now(),
              ),
              const PlanStep(
                index: 1,
                description: 'Create AuthUsecase',
                status: PlanStepStatus.inProgress,
              ),
              const PlanStep(
                index: 2,
                description: 'Write tests',
                status: PlanStepStatus.pending,
              ),
            ],
          ),
        ),
      ];

      await tester.pumpWidget(
        _buildChatScreen(chatRepoOverride: FakeChatRepository(messages)),
      );

      // Use pump() instead of pumpAndSettle() because PlanWidget renders
      // a CircularProgressIndicator for inProgress steps (infinite animation).
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Plan goal should be visible.
      expect(find.text('Refactor auth module'), findsOneWidget);

      // Steps should be rendered.
      expect(find.text('Extract TokenService'), findsOneWidget);
      expect(find.text('Create AuthUsecase'), findsOneWidget);
      expect(find.text('Write tests'), findsOneWidget);

      // Progress indicator text.
      expect(find.textContaining('33%'), findsOneWidget);
    });
  });
}

import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';

import '../../../helpers/fakes.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Fake [WsBridgeClient] that records calls and returns configurable results.
class _FakeWsBridgeClient implements WsBridgeClient {
  // --- sendNewTask ---
  SendCommandResult sendNewTaskResult = const SendCommandResult(success: true);
  final sendNewTaskCalls =
      <({String deviceToken, String sessionId, String task})>[];

  // --- sendAskUserReply ---
  SendCommandResult sendAskUserReplyResult = const SendCommandResult(
    success: true,
  );
  final sendAskUserReplyCalls =
      <
        ({String deviceToken, String sessionId, String question, String answer})
      >[];

  // --- cancelSession ---
  SendCommandResult cancelSessionResult = const SendCommandResult(
    success: true,
  );
  final cancelSessionCalls = <({String deviceToken, String sessionId})>[];

  // --- subscribeSession ---
  StreamController<SessionEvent>? sessionStreamController;

  @override
  Future<PingResult> ping() async {
    return PingResult(
      timestamp: DateTime.now(),
      serverName: 'Test Server',
      serverId: 'test-server-id',
    );
  }

  @override
  Stream<SessionEvent> subscribeSession({
    required String deviceToken,
    required String sessionId,
    String? lastEventId,
  }) {
    sessionStreamController ??= StreamController<SessionEvent>.broadcast();
    return sessionStreamController!.stream;
  }

  @override
  Future<SendCommandResult> sendNewTask({
    required String deviceToken,
    required String sessionId,
    required String task,
  }) async {
    sendNewTaskCalls.add((
      deviceToken: deviceToken,
      sessionId: sessionId,
      task: task,
    ));
    return sendNewTaskResult;
  }

  @override
  Future<SendCommandResult> sendAskUserReply({
    required String deviceToken,
    required String sessionId,
    required String question,
    required String answer,
  }) async {
    sendAskUserReplyCalls.add((
      deviceToken: deviceToken,
      sessionId: sessionId,
      question: question,
      answer: answer,
    ));
    return sendAskUserReplyResult;
  }

  @override
  Future<SendCommandResult> cancelSession({
    required String deviceToken,
    required String sessionId,
  }) async {
    cancelSessionCalls.add((deviceToken: deviceToken, sessionId: sessionId));
    return cancelSessionResult;
  }

  @override
  Future<void> dispose() async {}

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Fake WsConnection stub.
class _FakeWsConnection implements WsConnection {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Creates a test server with a device token.
Server _testServer({
  String id = 'srv-1',
  String name = 'Test Server',
  String deviceToken = 'test-token-123',
}) {
  return Server(
    id: id,
    name: name,
    bridgeUrl: 'ws://bridge:8080',
    isOnline: true,
    latencyMs: 10,
    pairedAt: DateTime(2026, 1, 1),
    deviceToken: deviceToken,
  );
}

/// Helper to create a [SessionEvent] with a given payload.
SessionEvent _event({
  String eventId = 'evt-1',
  String sessionId = 'session-1',
  SessionEventType type = SessionEventType.agentMessage,
  String agentId = 'agent-main',
  int step = 0,
  SessionEventPayload? payload,
}) {
  return SessionEvent(
    eventId: eventId,
    sessionId: sessionId,
    type: type,
    timestamp: DateTime(2026, 3, 1, 12, 0),
    agentId: agentId,
    step: step,
    payload: payload,
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('WsChatRepository', () {
    late FakeConnectionManager connectionManager;
    late _FakeWsBridgeClient fakeClient;
    late WsChatRepository repo;

    setUp(() {
      fakeClient = _FakeWsBridgeClient();
      connectionManager = FakeConnectionManager();
      repo = WsChatRepository(
        connectionManager: connectionManager,
        serverId: 'srv-1',
        sessionId: 'session-1',
      );
    });

    /// Adds a connected server to the fake connection manager.
    void addConnectedServer({Server? server}) {
      final s = server ?? _testServer();
      final conn = WsServerConnection(
        server: s,
        connection: _FakeWsConnection(),
        client: fakeClient,
      )..status = WsConnectionStatus.connected;
      connectionManager.addFakeConnection(s.id, conn);
    }

    tearDown(() async {
      repo.dispose();
      await fakeClient.sessionStreamController?.close();
      fakeClient.sessionStreamController = null;
    });

    group('initial state', () {
      test('getMessages returns empty list initially', () async {
        final messages = await repo.getMessages('session-1');

        expect(messages, isEmpty);
      });

      test('isSubscribed is false initially', () {
        expect(repo.isSubscribed, isFalse);
      });
    });

    group('sendMessage', () {
      test(
        'adds optimistic user message and delegates to connection manager',
        () async {
          addConnectedServer();

          final emittedMessages = <List<ChatMessage>>[];
          repo.watchMessages().listen(emittedMessages.add);

          await repo.sendMessage('session-1', 'Hello agent');

          // Optimistic message should be present.
          final messages = await repo.getMessages('session-1');
          expect(messages, hasLength(1));
          expect(messages.first.type, ChatMessageType.userMessage);
          expect(messages.first.content, 'Hello agent');

          // Client should have received the sendNewTask call.
          expect(fakeClient.sendNewTaskCalls, hasLength(1));
          expect(fakeClient.sendNewTaskCalls.first.task, 'Hello agent');
          expect(fakeClient.sendNewTaskCalls.first.sessionId, 'session-1');
        },
      );

      test('shows error message when sendNewTask fails', () async {
        addConnectedServer();
        fakeClient.sendNewTaskResult = const SendCommandResult(
          success: false,
          errorMessage: 'Connection lost',
        );

        final emitted = <List<ChatMessage>>[];
        repo.watchMessages().listen(emitted.add);

        await repo.sendMessage('session-1', 'Hello');

        final messages = await repo.getMessages('session-1');
        // Optimistic user message + error system message.
        expect(messages, hasLength(2));
        expect(messages[0].type, ChatMessageType.userMessage);
        expect(messages[1].type, ChatMessageType.systemMessage);
        expect(messages[1].content, contains('Failed to send'));
        expect(messages[1].content, contains('Connection lost'));
      });

      test(
        'shows default error when sendNewTask fails with empty message',
        () async {
          addConnectedServer();
          fakeClient.sendNewTaskResult = const SendCommandResult(
            success: false,
          );

          await repo.sendMessage('session-1', 'Hello');

          final messages = await repo.getMessages('session-1');
          expect(messages, hasLength(2));
          expect(messages[1].content, contains('Server not connected'));
        },
      );

      test('generates unique message IDs for each user message', () async {
        addConnectedServer();

        await repo.sendMessage('session-1', 'First');
        // Ensure at least 1 ms elapses so DateTime.now().millisecondsSinceEpoch
        // produces a different value for the second message ID.
        await Future<void>.delayed(const Duration(milliseconds: 2));
        await repo.sendMessage('session-1', 'Second');

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(2));
        expect(messages[0].id, isNot(messages[1].id));
      });
    });

    group('subscribe and event handling', () {
      setUp(() {
        // Add connected server so subscribeToSession returns a stream.
        addConnectedServer();
      });

      test('subscribe sets isSubscribed to true when stream is available', () {
        repo.subscribe();

        expect(repo.isSubscribed, isTrue);
      });

      test('handles AgentMessagePayload (complete)', () async {
        repo.subscribe();

        final emitted = <List<ChatMessage>>[];
        repo.watchMessages().listen(emitted.add);

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-agent-1',
            payload: const AgentMessagePayload(
              content: 'Hello from agent',
              isComplete: true,
            ),
          ),
        );

        // Give the stream a tick to process.
        await Future<void>.delayed(Duration.zero);

        expect(emitted, hasLength(1));
        expect(emitted.last, hasLength(1));
        expect(emitted.last.first.type, ChatMessageType.agentMessage);
        expect(emitted.last.first.content, 'Hello from agent');
      });

      test('handles streaming AgentMessagePayload chunks', () async {
        repo.subscribe();

        final emitted = <List<ChatMessage>>[];
        repo.watchMessages().listen(emitted.add);

        // First chunk.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-chunk-1',
            agentId: 'agent-1',
            step: 1,
            payload: const AgentMessagePayload(
              content: 'Hello ',
              isComplete: false,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        // Second chunk.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-chunk-2',
            agentId: 'agent-1',
            step: 1,
            payload: const AgentMessagePayload(
              content: 'world',
              isComplete: false,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        // There should be one message with accumulated content.
        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.content, 'Hello world');
      });

      test('handles ToolCallStartPayload', () async {
        repo.subscribe();

        final emitted = <List<ChatMessage>>[];
        repo.watchMessages().listen(emitted.add);

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-tc-start',
            payload: const ToolCallStartPayload(
              callId: 'call-1',
              toolName: 'read_file',
              arguments: {'path': '/src/main.dart'},
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.type, ChatMessageType.toolCall);
        expect(messages.first.toolCall, isNotNull);
        expect(messages.first.toolCall!.toolName, 'read_file');
        expect(messages.first.toolCall!.status, ToolCallStatus.running);
        expect(messages.first.toolCall!.arguments, {'path': '/src/main.dart'});
      });

      test('handles ToolCallEndPayload (success)', () async {
        repo.subscribe();

        // First start the tool call.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-tc-start',
            payload: const ToolCallStartPayload(
              callId: 'call-1',
              toolName: 'read_file',
              arguments: {'path': '/src/main.dart'},
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        // Then end it.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-tc-end',
            payload: const ToolCallEndPayload(
              callId: 'call-1',
              toolName: 'read_file',
              resultSummary: '50 lines',
              hasError: false,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.toolCall!.status, ToolCallStatus.completed);
        expect(messages.first.toolCall!.result, '50 lines');
        expect(messages.first.toolCall!.error, isNull);
      });

      test('handles ToolCallEndPayload (error)', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-tc-start',
            payload: const ToolCallStartPayload(
              callId: 'call-2',
              toolName: 'execute',
              arguments: {'cmd': 'rm -rf /'},
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-tc-end',
            payload: const ToolCallEndPayload(
              callId: 'call-2',
              toolName: 'execute',
              resultSummary: 'Permission denied',
              hasError: true,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages.first.toolCall!.status, ToolCallStatus.failed);
        expect(messages.first.toolCall!.error, 'Permission denied');
      });

      test('handles ReasoningPayload (complete only)', () async {
        repo.subscribe();

        // Incomplete reasoning should be ignored.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-reason-incomplete',
            payload: const ReasoningPayload(
              content: 'Thinking...',
              isComplete: false,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        var messages = await repo.getMessages('session-1');
        expect(messages, isEmpty);

        // Complete reasoning should produce a message.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-reason-complete',
            payload: const ReasoningPayload(
              content: 'I need to analyze the structure',
              isComplete: true,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.type, ChatMessageType.reasoning);
        expect(messages.first.content, 'I need to analyze the structure');
      });

      test('handles AskUserPayload', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-ask-1',
            payload: const AskUserPayload(
              question: 'Which framework?',
              options: ['React', 'Vue', 'Angular'],
              isAnswered: false,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.type, ChatMessageType.askUser);
        expect(messages.first.askUser, isNotNull);
        expect(messages.first.askUser!.question, 'Which framework?');
        expect(messages.first.askUser!.options, ['React', 'Vue', 'Angular']);
        expect(messages.first.askUser!.status, AskUserStatus.pending);
      });

      test('ignores already-answered AskUserPayload', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-ask-answered',
            payload: const AskUserPayload(
              question: 'Already answered',
              options: [],
              isAnswered: true,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, isEmpty);
      });

      test('handles PlanPayload', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-plan-1',
            agentId: 'agent-main',
            payload: const PlanPayload(
              planName: 'Refactor module',
              steps: [
                PlanStepPayload(
                  title: 'Analyze code',
                  status: WsPlanStepStatus.completed,
                ),
                PlanStepPayload(
                  title: 'Write tests',
                  status: WsPlanStepStatus.inProgress,
                ),
                PlanStepPayload(
                  title: 'Refactor',
                  status: WsPlanStepStatus.pending,
                ),
              ],
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.type, ChatMessageType.planUpdate);
        expect(messages.first.plan, isNotNull);
        expect(messages.first.plan!.goal, 'Refactor module');
        expect(messages.first.plan!.steps, hasLength(3));
        expect(messages.first.plan!.steps[0].status, PlanStepStatus.completed);
        expect(messages.first.plan!.steps[1].status, PlanStepStatus.inProgress);
        expect(messages.first.plan!.steps[2].status, PlanStepStatus.pending);
      });

      test('handles SessionStatusPayload with custom message', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-status-1',
            payload: const SessionStatusPayload(
              state: MobileSessionState.completed,
              message: 'Task finished successfully',
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.type, ChatMessageType.systemMessage);
        expect(messages.first.content, 'Task finished successfully');
      });

      test(
        'handles SessionStatusPayload with empty message (uses default)',
        () async {
          repo.subscribe();

          fakeClient.sessionStreamController!.add(
            _event(
              eventId: 'evt-status-2',
              payload: const SessionStatusPayload(
                state: MobileSessionState.active,
                message: '',
              ),
            ),
          );
          await Future<void>.delayed(Duration.zero);

          final messages = await repo.getMessages('session-1');
          expect(messages.first.content, 'Session status: active');
        },
      );

      test('handles ErrorPayload', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-error-1',
            payload: const ErrorPayload(
              code: 'RATE_LIMIT',
              message: 'Too many requests',
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
        expect(messages.first.type, ChatMessageType.systemMessage);
        expect(messages.first.content, 'Error [RATE_LIMIT]: Too many requests');
      });

      test('ignores event with null payload', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(eventId: 'evt-null', payload: null),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, isEmpty);
      });

      test('tracks lastEventId from received events', () async {
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-100',
            payload: const AgentMessagePayload(
              content: 'msg',
              isComplete: true,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages, hasLength(1));
      });
    });

    group('answerAskUser', () {
      test(
        'updates message status and sends reply via connection manager',
        () async {
          addConnectedServer();

          // First inject an ask-user message via event stream.
          repo.subscribe();

          fakeClient.sessionStreamController!.add(
            _event(
              eventId: 'evt-ask-reply',
              payload: const AskUserPayload(
                question: 'Continue?',
                options: ['Yes', 'No'],
                isAnswered: false,
              ),
            ),
          );
          await Future<void>.delayed(Duration.zero);

          final askUserId = (await repo.getMessages(
            'session-1',
          )).first.askUser!.id;

          await repo.answerAskUser('session-1', askUserId, 'Yes');

          // Message should be updated optimistically.
          final messages = await repo.getMessages('session-1');
          expect(messages.first.askUser!.status, AskUserStatus.answered);
          expect(messages.first.askUser!.answer, 'Yes');

          // Client call should have been made.
          expect(fakeClient.sendAskUserReplyCalls, hasLength(1));
          expect(fakeClient.sendAskUserReplyCalls.first.question, 'Continue?');
          expect(fakeClient.sendAskUserReplyCalls.first.answer, 'Yes');
        },
      );

      test('shows error message when sendAskUserReply fails', () async {
        addConnectedServer();
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-ask-err',
            payload: const AskUserPayload(
              question: 'Proceed?',
              options: ['Yes', 'No'],
              isAnswered: false,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final askUserId = (await repo.getMessages(
          'session-1',
        )).first.askUser!.id;

        fakeClient.sendAskUserReplyResult = const SendCommandResult(
          success: false,
          errorMessage: 'No device token',
        );

        await repo.answerAskUser('session-1', askUserId, 'Yes');

        final messages = await repo.getMessages('session-1');
        // Ask-user message (updated) + error system message.
        expect(messages, hasLength(2));
        expect(messages[1].type, ChatMessageType.systemMessage);
        expect(messages[1].content, contains('Failed to send reply'));
        expect(messages[1].content, contains('No device token'));
      });
    });

    group('cancel', () {
      test('delegates to connection manager', () async {
        addConnectedServer();

        await repo.cancel('session-1');

        expect(fakeClient.cancelSessionCalls, hasLength(1));
        expect(fakeClient.cancelSessionCalls.first.sessionId, 'session-1');
      });
    });

    group('watchMessages', () {
      test('emits message lists on events', () async {
        addConnectedServer();
        repo.subscribe();

        final collected = <List<ChatMessage>>[];
        final sub = repo.watchMessages().listen(collected.add);

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-watch-1',
            payload: const AgentMessagePayload(
              content: 'First',
              isComplete: true,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-watch-2',
            payload: const AgentMessagePayload(
              content: 'Second',
              isComplete: true,
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        expect(collected, hasLength(2));
        expect(collected[0], hasLength(1));
        expect(collected[1], hasLength(2));

        await sub.cancel();
      });
    });

    group('dispose', () {
      test('closes stream controllers without error', () {
        // Double dispose should not throw.
        repo.dispose();
        expect(() => repo.dispose(), returnsNormally);
      });
    });

    group('upsert behavior', () {
      test('replaces existing message with same id', () async {
        addConnectedServer();
        repo.subscribe();

        // Plan messages use id 'plan-<agentId>', so sending two plans
        // for the same agent should result in one message.
        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-plan-first',
            agentId: 'agent-1',
            payload: const PlanPayload(
              planName: 'Plan v1',
              steps: [
                PlanStepPayload(
                  title: 'Step 1',
                  status: WsPlanStepStatus.pending,
                ),
              ],
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-plan-second',
            agentId: 'agent-1',
            payload: const PlanPayload(
              planName: 'Plan v2',
              steps: [
                PlanStepPayload(
                  title: 'Step 1',
                  status: WsPlanStepStatus.completed,
                ),
                PlanStepPayload(
                  title: 'Step 2',
                  status: WsPlanStepStatus.inProgress,
                ),
              ],
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        // Both plans used id 'plan-agent-1', so only one message.
        expect(messages, hasLength(1));
        expect(messages.first.plan!.goal, 'Plan v2');
        expect(messages.first.plan!.steps, hasLength(2));
      });
    });

    group('plan step status mapping', () {
      test('maps WsPlanStepStatus.failed to PlanStepStatus.failed', () async {
        addConnectedServer();
        repo.subscribe();

        fakeClient.sessionStreamController!.add(
          _event(
            eventId: 'evt-plan-failed-step',
            agentId: 'agent-1',
            payload: const PlanPayload(
              planName: 'Plan with failed step',
              steps: [
                PlanStepPayload(
                  title: 'Failed step',
                  status: WsPlanStepStatus.failed,
                ),
                PlanStepPayload(
                  title: 'Unspecified step',
                  status: WsPlanStepStatus.unspecified,
                ),
              ],
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);

        final messages = await repo.getMessages('session-1');
        expect(messages.first.plan!.steps[0].status, PlanStepStatus.failed);
        expect(messages.first.plan!.steps[1].status, PlanStepStatus.pending);
      });
    });
  });
}

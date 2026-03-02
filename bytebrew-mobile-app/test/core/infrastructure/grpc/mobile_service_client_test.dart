import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart';

import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/mobile_service_client.dart';

void main() {
  group('MobileServiceClient', () {
    test('constructor creates client without errors', () {
      // Create a channel pointing to a dummy address (we will not
      // actually make any RPC calls).
      final channel = ClientChannel(
        'localhost',
        port: 60401,
        options: const ChannelOptions(
          credentials: ChannelCredentials.insecure(),
        ),
      );

      final client = MobileServiceClient(channel: channel);

      expect(client, isNotNull);

      // Clean up the channel.
      channel.shutdown();
    });

    test('close disposes channel', () async {
      final channel = ClientChannel(
        'localhost',
        port: 60401,
        options: const ChannelOptions(
          credentials: ChannelCredentials.insecure(),
        ),
      );

      final client = MobileServiceClient(channel: channel);

      // close() should call channel.shutdown() and complete without error.
      await expectLater(client.close(), completes);
    });

    test('multiple close calls do not throw', () async {
      final channel = ClientChannel(
        'localhost',
        port: 60401,
        options: const ChannelOptions(
          credentials: ChannelCredentials.insecure(),
        ),
      );

      final client = MobileServiceClient(channel: channel);

      // First close should work.
      await expectLater(client.close(), completes);

      // Second close on an already-shutdown channel should also not throw.
      // (ClientChannel.shutdown is idempotent.)
      await expectLater(client.close(), completes);
    });
  });

  group('DTOs', () {
    test('PairResult stores all fields', () {
      final result = PairResult(
        deviceId: 'dev-1',
        deviceToken: 'token-abc',
        serverName: 'My Server',
        serverId: 'srv-1',
      );

      expect(result.deviceId, 'dev-1');
      expect(result.deviceToken, 'token-abc');
      expect(result.serverName, 'My Server');
      expect(result.serverId, 'srv-1');
      expect(result.serverPublicKey, isNull);
    });

    test('PingResult stores timestamp and server info', () {
      final now = DateTime.now();
      final result = PingResult(
        timestamp: now,
        serverName: 'Test Server',
        serverId: 'test-id',
      );

      expect(result.timestamp, now);
      expect(result.serverName, 'Test Server');
      expect(result.serverId, 'test-id');
    });

    test('SendCommandResult default errorMessage is empty', () {
      const result = SendCommandResult(success: true);

      expect(result.success, isTrue);
      expect(result.errorMessage, '');
    });

    test('SendCommandResult stores error message', () {
      const result = SendCommandResult(
        success: false,
        errorMessage: 'Connection lost',
      );

      expect(result.success, isFalse);
      expect(result.errorMessage, 'Connection lost');
    });

    test('SessionEvent stores all fields', () {
      final now = DateTime.now();
      final event = SessionEvent(
        eventId: 'evt-1',
        sessionId: 'sess-1',
        type: SessionEventType.agentMessage,
        timestamp: now,
        agentId: 'agent-1',
        step: 3,
        payload: const AgentMessagePayload(
          content: 'Hello',
          isComplete: true,
        ),
      );

      expect(event.eventId, 'evt-1');
      expect(event.sessionId, 'sess-1');
      expect(event.type, SessionEventType.agentMessage);
      expect(event.timestamp, now);
      expect(event.agentId, 'agent-1');
      expect(event.step, 3);
      expect(event.payload, isA<AgentMessagePayload>());

      final payload = event.payload! as AgentMessagePayload;
      expect(payload.content, 'Hello');
      expect(payload.isComplete, isTrue);
    });

    test('SessionEvent defaults', () {
      final event = SessionEvent(
        eventId: 'evt-2',
        sessionId: 'sess-2',
        type: SessionEventType.unspecified,
        timestamp: DateTime.now(),
      );

      expect(event.agentId, '');
      expect(event.step, 0);
      expect(event.payload, isNull);
    });

    group('SessionEventPayload subtypes', () {
      test('ToolCallStartPayload', () {
        const payload = ToolCallStartPayload(
          callId: 'tc-1',
          toolName: 'read_file',
          arguments: {'path': '/src/main.dart'},
        );

        expect(payload.callId, 'tc-1');
        expect(payload.toolName, 'read_file');
        expect(payload.arguments, {'path': '/src/main.dart'});
      });

      test('ToolCallEndPayload', () {
        const payload = ToolCallEndPayload(
          callId: 'tc-1',
          toolName: 'read_file',
          resultSummary: '50 lines',
          hasError: false,
        );

        expect(payload.callId, 'tc-1');
        expect(payload.resultSummary, '50 lines');
        expect(payload.hasError, isFalse);
      });

      test('ReasoningPayload', () {
        const payload = ReasoningPayload(
          content: 'Analyzing...',
          isComplete: false,
        );

        expect(payload.content, 'Analyzing...');
        expect(payload.isComplete, isFalse);
      });

      test('AskUserPayload', () {
        const payload = AskUserPayload(
          question: 'Which framework?',
          options: ['React', 'Vue'],
          isAnswered: false,
        );

        expect(payload.question, 'Which framework?');
        expect(payload.options, ['React', 'Vue']);
        expect(payload.isAnswered, isFalse);
      });

      test('PlanPayload with steps', () {
        const payload = PlanPayload(
          planName: 'Refactoring',
          steps: [
            PlanStepPayload(
              title: 'Analyze code',
              status: PlanStepStatus.completed,
            ),
            PlanStepPayload(
              title: 'Write tests',
              status: PlanStepStatus.pending,
            ),
          ],
        );

        expect(payload.planName, 'Refactoring');
        expect(payload.steps, hasLength(2));
        expect(payload.steps[0].status, PlanStepStatus.completed);
        expect(payload.steps[1].status, PlanStepStatus.pending);
      });

      test('SessionStatusPayload', () {
        const payload = SessionStatusPayload(
          state: MobileSessionState.active,
          message: 'Processing task',
        );

        expect(payload.state, MobileSessionState.active);
        expect(payload.message, 'Processing task');
      });

      test('ErrorPayload', () {
        const payload = ErrorPayload(
          code: 'TIMEOUT',
          message: 'Request timed out',
        );

        expect(payload.code, 'TIMEOUT');
        expect(payload.message, 'Request timed out');
      });
    });

    test('ListSessionsResult stores sessions', () {
      final result = ListSessionsResult(
        sessions: [
          MobileSession(
            sessionId: 's-1',
            projectKey: 'proj-1',
            projectRoot: '/home/user/project',
            status: MobileSessionState.active,
            currentTask: 'Refactor code',
            startedAt: DateTime.now(),
            lastActivityAt: DateTime.now(),
            hasAskUser: false,
            platform: 'linux',
          ),
        ],
        serverName: 'Dev Server',
        serverId: 'srv-1',
      );

      expect(result.sessions, hasLength(1));
      expect(result.sessions.first.sessionId, 's-1');
      expect(result.serverName, 'Dev Server');
    });
  });
}

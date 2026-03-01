import 'dart:async';
import 'dart:convert';

import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// ---------------------------------------------------------------------------
// Fakes (same pattern as ws_chat_repository_test.dart)
// ---------------------------------------------------------------------------

class FakeWebSocketSink implements WebSocketSink {
  final List<String> sent = [];
  bool isClosed = false;

  @override
  void add(dynamic data) => sent.add(data.toString());

  @override
  void addError(Object error, [StackTrace? stackTrace]) {}

  @override
  Future<dynamic> addStream(Stream<dynamic> stream) => Future.value();

  @override
  Future<dynamic> close([int? closeCode, String? closeReason]) {
    isClosed = true;
    return Future.value();
  }

  @override
  Future<dynamic> get done => Future.value();
}

class FakeWebSocketChannel implements WebSocketChannel {
  FakeWebSocketChannel()
    : _incoming = StreamController<dynamic>.broadcast(),
      sink = FakeWebSocketSink();

  final StreamController<dynamic> _incoming;

  @override
  final FakeWebSocketSink sink;

  @override
  Stream<dynamic> get stream => _incoming.stream;

  @override
  Future<void> get ready => Future.value();

  @override
  int? get closeCode => null;

  @override
  String? get closeReason => null;

  @override
  String? get protocol => null;

  /// Simulates incoming data from the server.
  void receive(String data) => _incoming.add(data);

  /// Closes the incoming stream to simulate a disconnection.
  Future<void> closeIncoming() => _incoming.close();

  @override
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Fixed timestamp used across tests to keep assertions deterministic.
final _fixedTimestampMs = DateTime(2026, 3, 1, 12, 0).millisecondsSinceEpoch;

/// Creates a repository with a [FakeWebSocketChannel] and connects it.
///
/// Returns both so the caller can inject events through the channel.
Future<({WsChatRepository repo, FakeWebSocketChannel channel})>
_createConnectedRepo() async {
  final channel = FakeWebSocketChannel();
  final repo = WsChatRepository(
    wsUrl: 'ws://localhost:8765',
    channelFactory: (_) => channel,
  );
  await repo.connect();
  return (repo: repo, channel: channel);
}

/// Sends an empty init payload to initialise the repository.
void _sendEmptyInit(FakeWebSocketChannel channel) {
  channel.receive(
    jsonEncode({'type': 'init', 'messages': <dynamic>[]}),
  );
}

/// Sends an AgentLifecycle event through the channel.
void _sendAgentLifecycle(
  FakeWebSocketChannel channel, {
  required String agentId,
  required String lifecycleType,
  String? description,
}) {
  channel.receive(
    jsonEncode({
      'type': 'event',
      'event': {
        'type': 'AgentLifecycle',
        'agentId': agentId,
        'lifecycleType': lifecycleType,
        'description': description ?? agentId,
      },
    }),
  );
}

/// Sends a ToolExecutionStarted event with an optional agentId.
void _sendToolStarted(
  FakeWebSocketChannel channel, {
  required String callId,
  required String toolName,
  String? agentId,
  Map<String, String>? arguments,
}) {
  channel.receive(
    jsonEncode({
      'type': 'event',
      'event': {
        'type': 'ToolExecutionStarted',
        'execution': {
          'callId': callId,
          'toolName': toolName,
          'arguments': arguments ?? <String, String>{},
          if (agentId != null) 'agentId': agentId,
        },
      },
    }),
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('WsChatRepository agent tracking', () {
    // -----------------------------------------------------------------------
    // agent_spawned -> agent appears in watchAgents
    // -----------------------------------------------------------------------
    test('agent_spawned adds agent to watchAgents stream', () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      final agentsFuture = repo.watchAgents().first;

      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_spawned',
        description: 'Code Writer',
      );

      final agents = await agentsFuture;

      expect(agents, hasLength(1));
      expect(agents.first.agentId, 'agent-coder');
      expect(agents.first.status, AgentStatus.running);
      expect(agents.first.description, 'Code Writer');
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // agent_completed -> agent status updated
    // -----------------------------------------------------------------------
    test('agent_completed updates agent status to completed', () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Spawn agent first.
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_spawned',
        description: 'Code Writer',
      );
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Now complete it.
      final agentsFuture = repo.watchAgents().first;
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_completed',
        description: 'Code Writer',
      );

      final agents = await agentsFuture;

      expect(agents, hasLength(1));
      expect(agents.first.agentId, 'agent-coder');
      expect(agents.first.status, AgentStatus.completed);
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // agent_failed -> agent status updated
    // -----------------------------------------------------------------------
    test('agent_failed updates agent status to failed', () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Spawn agent first.
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-reviewer',
        lifecycleType: 'agent_spawned',
        description: 'Code Reviewer',
      );
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Now fail it.
      final agentsFuture = repo.watchAgents().first;
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-reviewer',
        lifecycleType: 'agent_failed',
        description: 'Code Reviewer',
      );

      final agents = await agentsFuture;

      expect(agents, hasLength(1));
      expect(agents.first.agentId, 'agent-reviewer');
      expect(agents.first.status, AgentStatus.failed);
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // Lifecycle event -> system message in messages stream
    // -----------------------------------------------------------------------
    test('lifecycle event inserts system message into messages stream',
        () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      final messagesFuture = repo.watchMessages().first;
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_spawned',
        description: 'Code Writer',
      );

      final messages = await messagesFuture;

      // Should contain at least the system message for the lifecycle event.
      final systemMessages = messages.where(
        (m) => m.type == ChatMessageType.systemMessage,
      );
      expect(systemMessages, isNotEmpty);

      final lifecycleMsg = systemMessages.first;
      expect(lifecycleMsg.content, contains('Agent started'));
      expect(lifecycleMsg.content, contains('Code Writer'));
      expect(lifecycleMsg.agentId, 'agent-coder');
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // agent_completed lifecycle -> system message content
    // -----------------------------------------------------------------------
    test('agent_completed lifecycle inserts completion system message',
        () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Spawn first.
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_spawned',
        description: 'Code Writer',
      );
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Complete.
      final messagesFuture = repo.watchMessages().first;
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_completed',
        description: 'Code Writer',
      );

      final messages = await messagesFuture;
      final completionMsgs = messages.where(
        (m) =>
            m.type == ChatMessageType.systemMessage &&
            m.content.contains('Agent completed'),
      );
      expect(completionMsgs, isNotEmpty);
      expect(completionMsgs.first.content, contains('Code Writer'));
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // ToolExecutionStarted with agentId -> ChatMessage.agentId set
    // -----------------------------------------------------------------------
    test('ToolExecutionStarted with agentId sets ChatMessage.agentId',
        () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      final messagesFuture = repo.watchMessages().first;
      _sendToolStarted(
        channel,
        callId: 'tc-1',
        toolName: 'read_file',
        agentId: 'agent-coder',
        arguments: {'path': '/src/main.dart'},
      );

      final messages = await messagesFuture;

      final toolMsg = messages.where((m) => m.id == 'tc-1').first;
      expect(toolMsg.type, ChatMessageType.toolCall);
      expect(toolMsg.agentId, 'agent-coder');
      expect(toolMsg.toolCall!.toolName, 'read_file');
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // ToolExecutionStarted without agentId -> ChatMessage.agentId null
    // -----------------------------------------------------------------------
    test('ToolExecutionStarted without agentId leaves agentId null', () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      final messagesFuture = repo.watchMessages().first;
      _sendToolStarted(
        channel,
        callId: 'tc-2',
        toolName: 'search_code',
      );

      final messages = await messagesFuture;

      final toolMsg = messages.where((m) => m.id == 'tc-2').first;
      expect(toolMsg.agentId, isNull);
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // Init with messages containing agentId -> agents list populated
    // -----------------------------------------------------------------------
    test('init with agentId messages populates agents list', () async {
      final (:repo, :channel) = await _createConnectedRepo();

      final agentsFuture = repo.watchAgents().first;

      channel.receive(
        jsonEncode({
          'type': 'init',
          'messages': [
            {
              'id': 'msg-1',
              'role': 'user',
              'content': 'Hello',
              'timestamp': _fixedTimestampMs,
            },
            {
              'id': 'msg-2',
              'role': 'assistant',
              'content': 'Working on it',
              'timestamp': _fixedTimestampMs,
              'agentId': 'agent-coder',
            },
            {
              'id': 'msg-3',
              'role': 'tool',
              'content': '',
              'timestamp': _fixedTimestampMs,
              'agentId': 'agent-coder',
              'toolCall': {
                'id': 'tc-1',
                'toolName': 'read_file',
                'arguments': {'path': '/tmp'},
                'status': 'completed',
              },
            },
            {
              'id': 'msg-4',
              'role': 'assistant',
              'content': 'Reviewing',
              'timestamp': _fixedTimestampMs,
              'agentId': 'agent-reviewer',
            },
          ],
        }),
      );

      final agents = await agentsFuture;

      // Should detect 2 unique agent IDs (not counting 'supervisor').
      expect(agents, hasLength(2));
      final agentIds = agents.map((a) => a.agentId).toSet();
      expect(agentIds, contains('agent-coder'));
      expect(agentIds, contains('agent-reviewer'));
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // Init with supervisor agentId -> not added to agents
    // -----------------------------------------------------------------------
    test('init with supervisor agentId does not add to agents', () async {
      final (:repo, :channel) = await _createConnectedRepo();

      channel.receive(
        jsonEncode({
          'type': 'init',
          'messages': [
            {
              'id': 'msg-1',
              'role': 'assistant',
              'content': 'Coordinating',
              'timestamp': _fixedTimestampMs,
              'agentId': 'supervisor',
            },
            {
              'id': 'msg-2',
              'role': 'assistant',
              'content': 'Working',
              'timestamp': _fixedTimestampMs,
              'agentId': 'agent-coder',
            },
          ],
        }),
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      final messages = await repo.getMessages('any');
      expect(messages, hasLength(2));

      // Only agent-coder should be in agents, not supervisor.
      final agentsFuture = repo.watchAgents().first;
      // Trigger another lifecycle to get current snapshot via stream.
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_completed',
        description: 'agent-coder',
      );

      final agents = await agentsFuture;
      final agentIds = agents.map((a) => a.agentId).toSet();
      expect(agentIds, contains('agent-coder'));
      expect(agentIds, isNot(contains('supervisor')));
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // Multiple agents tracked simultaneously
    // -----------------------------------------------------------------------
    test('tracks multiple agents simultaneously', () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_spawned',
        description: 'Code Writer',
      );
      await Future<void>.delayed(const Duration(milliseconds: 50));

      final agentsFuture = repo.watchAgents().first;
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-reviewer',
        lifecycleType: 'agent_spawned',
        description: 'Code Reviewer',
      );

      final agents = await agentsFuture;

      expect(agents, hasLength(2));
      final agentIds = agents.map((a) => a.agentId).toSet();
      expect(agentIds, containsAll(['agent-coder', 'agent-reviewer']));
      repo.dispose();
    });

    // -----------------------------------------------------------------------
    // Second init clears agents
    // -----------------------------------------------------------------------
    test('second init clears agents from previous session', () async {
      final (:repo, :channel) = await _createConnectedRepo();
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Spawn an agent.
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-coder',
        lifecycleType: 'agent_spawned',
        description: 'Code Writer',
      );
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Second init should clear agents.
      _sendEmptyInit(channel);
      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Spawn a new agent to trigger emission and verify old ones are gone.
      final agentsFuture = repo.watchAgents().first;
      _sendAgentLifecycle(
        channel,
        agentId: 'agent-new',
        lifecycleType: 'agent_spawned',
        description: 'New Agent',
      );

      final agents = await agentsFuture;
      expect(agents, hasLength(1));
      expect(agents.first.agentId, 'agent-new');
      repo.dispose();
    });
  });
}

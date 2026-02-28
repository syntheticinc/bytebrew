import 'dart:async';
import 'dart:convert';

import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// ---------------------------------------------------------------------------
// Fakes
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

  // -- StreamChannelMixin overrides (delegate to default no-ops) --

  @override
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Fixed timestamp used across tests to keep assertions deterministic.
final _fixedTimestampMs = DateTime(2026, 3, 1, 12, 0).millisecondsSinceEpoch;

/// Fixed ISO 8601 timestamp string matching CLI output format.
const _fixedTimestampIso = '2026-02-28T14:46:03.990Z';

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

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('WsChatRepository', () {
    // -----------------------------------------------------------------------
    // TC-13: Connection status
    // -----------------------------------------------------------------------
    group('connect', () {
      test(
        'TC-13a: emits true on connectionStatus when connected',
        () async {
          final channel = FakeWebSocketChannel();
          final repo = WsChatRepository(
            wsUrl: 'ws://localhost:8765',
            channelFactory: (_) => channel,
          );

          final statusFuture = repo.connectionStatus.first;
          await repo.connect();

          expect(await statusFuture, isTrue);
          repo.dispose();
        },
      );

      test(
        'TC-13b: emits false on connectionStatus when channel closes',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          final statusFuture = repo.connectionStatus.first;
          await channel.closeIncoming();
          await Future<void>.delayed(const Duration(milliseconds: 50));

          expect(await statusFuture, isFalse);
          repo.dispose();
        },
      );
    });

    // -----------------------------------------------------------------------
    // TC-1: Init payload loads messages
    // -----------------------------------------------------------------------
    group('TC-1: init payload loads messages', () {
      test('parses 3 messages and emits through watchMessages', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

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
                'content': 'Hi there',
                'timestamp': _fixedTimestampMs,
              },
              {
                'id': 'msg-3',
                'role': 'assistant',
                'content': 'How can I help?',
                'timestamp': _fixedTimestampMs,
              },
            ],
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(3));
        expect(messages[0].id, 'msg-1');
        expect(messages[0].type, ChatMessageType.userMessage);
        expect(messages[0].content, 'Hello');
        expect(messages[1].id, 'msg-2');
        expect(messages[1].type, ChatMessageType.agentMessage);
        expect(messages[1].content, 'Hi there');
        expect(messages[2].id, 'msg-3');
        expect(messages[2].type, ChatMessageType.agentMessage);
        expect(messages[2].content, 'How can I help?');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-2: Init clears previous messages
    // -----------------------------------------------------------------------
    group('TC-2: init clears previous messages', () {
      test('second init replaces first init messages', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // First init with 2 messages.
        channel.receive(
          jsonEncode({
            'type': 'init',
            'messages': [
              {
                'id': 'msg-1',
                'role': 'user',
                'content': 'First',
                'timestamp': _fixedTimestampMs,
              },
              {
                'id': 'msg-2',
                'role': 'assistant',
                'content': 'Second',
                'timestamp': _fixedTimestampMs,
              },
            ],
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Second init with 1 message -- should replace, not append.
        final messagesFuture = repo.watchMessages().first;
        channel.receive(
          jsonEncode({
            'type': 'init',
            'messages': [
              {
                'id': 'msg-3',
                'role': 'user',
                'content': 'Only one',
                'timestamp': _fixedTimestampMs,
              },
            ],
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].id, 'msg-3');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-3: MessageCompleted adds new message
    // -----------------------------------------------------------------------
    group('TC-3: MessageCompleted adds new message', () {
      test('adds message to empty list', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Start with empty init.
        channel.receive(
          jsonEncode({'type': 'init', 'messages': <dynamic>[]}),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-3',
                'role': 'assistant',
                'content': 'Completed message',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].id, 'msg-3');
        expect(messages[0].content, 'Completed message');
        expect(messages[0].type, ChatMessageType.agentMessage);
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-4: MessageCompleted updates existing message (same ID)
    // -----------------------------------------------------------------------
    group('TC-4: MessageCompleted updates existing (same ID)', () {
      test('same id updates content, no duplication', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // First version.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-upsert',
                'role': 'assistant',
                'content': 'Version 1',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Updated version with same id.
        final messagesFuture = repo.watchMessages().first;
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-upsert',
                'role': 'assistant',
                'content': 'Version 2',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );

        final messages = await messagesFuture;
        expect(messages, hasLength(1));
        expect(messages[0].content, 'Version 2');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-5: User message optimistic update -- NO DUPLICATION (CRITICAL)
    // -----------------------------------------------------------------------
    group('TC-5: user message dedup (CRITICAL)', () {
      test(
        'server MessageCompleted replaces optimistic user message',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          // Init empty.
          channel.receive(
            jsonEncode({'type': 'init', 'messages': <dynamic>[]}),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          // User sends message (optimistic update created with id=user-*).
          await repo.sendMessage('session', 'Hello');
          await Future<void>.delayed(const Duration(milliseconds: 50));

          // Verify optimistic message exists.
          final afterSend = await repo.getMessages('session');
          expect(afterSend, hasLength(1));
          expect(afterSend[0].id, startsWith('user-'));
          expect(afterSend[0].content, 'Hello');

          // Server sends MessageCompleted with different ID, same content.
          final messagesFuture = repo.watchMessages().first;
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'MessageCompleted',
                'message': {
                  'id': 'a7b4c3d2-uuid',
                  'role': 'user',
                  'content': 'Hello',
                  'timestamp': _fixedTimestampIso,
                },
              },
            }),
          );

          final messages = await messagesFuture;

          // CRITICAL: exactly 1 user message, not 2.
          expect(messages, hasLength(1));
          // The server version replaces the optimistic one.
          expect(messages[0].id, 'a7b4c3d2-uuid');
          expect(messages[0].content, 'Hello');
          expect(messages[0].type, ChatMessageType.userMessage);
          repo.dispose();
        },
      );
    });

    // -----------------------------------------------------------------------
    // TC-6: User message -- different content doesn't match
    // -----------------------------------------------------------------------
    group('TC-6: user message dedup -- different content keeps both', () {
      test(
        'server message with different content does not replace optimistic',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          // Init empty.
          channel.receive(
            jsonEncode({'type': 'init', 'messages': <dynamic>[]}),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          // Optimistic user message.
          await repo.sendMessage('session', 'Hello');
          await Future<void>.delayed(const Duration(milliseconds: 50));

          // Server sends a user message with DIFFERENT content.
          final messagesFuture = repo.watchMessages().first;
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'MessageCompleted',
                'message': {
                  'id': 'uuid-different',
                  'role': 'user',
                  'content': 'Different text',
                  'timestamp': _fixedTimestampIso,
                },
              },
            }),
          );

          final messages = await messagesFuture;

          // Both messages kept -- they are genuinely different.
          expect(messages, hasLength(2));
          expect(messages[0].content, 'Hello');
          expect(messages[1].content, 'Different text');
          repo.dispose();
        },
      );
    });

    // -----------------------------------------------------------------------
    // TC-7: Agent message received
    // -----------------------------------------------------------------------
    group('TC-7: agent message received', () {
      test('MessageCompleted with role assistant appears correctly', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'agent-msg-1',
                'role': 'assistant',
                'content': 'I can help with that!',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].id, 'agent-msg-1');
        expect(messages[0].type, ChatMessageType.agentMessage);
        expect(messages[0].content, 'I can help with that!');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-8: ToolExecutionStarted creates tool call message
    // -----------------------------------------------------------------------
    group('TC-8: ToolExecutionStarted', () {
      test('creates running tool message', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionStarted',
              'execution': {
                'callId': 'tc-1',
                'toolName': 'Read',
                'arguments': {'path': '/foo'},
              },
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        final msg = messages[0];
        expect(msg.id, 'tc-1');
        expect(msg.type, ChatMessageType.toolCall);
        expect(msg.toolCall, isNotNull);
        expect(msg.toolCall!.toolName, 'Read');
        expect(msg.toolCall!.arguments, {'path': '/foo'});
        expect(msg.toolCall!.status, ToolCallStatus.running);
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-9: ToolExecutionCompleted updates tool call status
    // -----------------------------------------------------------------------
    group('TC-9: ToolExecutionCompleted', () {
      test('updates tool from running to completed', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Start tool.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionStarted',
              'execution': {
                'callId': 'tc-1',
                'toolName': 'Read',
                'arguments': {'path': '/foo'},
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Complete tool.
        final messagesFuture = repo.watchMessages().first;
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionCompleted',
              'execution': {'callId': 'tc-1', 'result': 'done'},
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        final msg = messages[0];
        expect(msg.id, 'tc-1');
        expect(msg.toolCall!.status, ToolCallStatus.completed);
        expect(msg.toolCall!.result, 'done');
        repo.dispose();
      });

      test('sets status to failed when error is present', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Start.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionStarted',
              'execution': {
                'callId': 'tc-err',
                'toolName': 'execute',
                'arguments': <String, dynamic>{},
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Complete with error.
        final messagesFuture = repo.watchMessages().first;
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionCompleted',
              'execution': {'callId': 'tc-err', 'error': 'command failed'},
            },
          }),
        );

        final messages = await messagesFuture;
        final msg = messages[0];
        expect(msg.toolCall!.status, ToolCallStatus.failed);
        expect(msg.toolCall!.error, 'command failed');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-10: AskUserRequested creates ask-user message
    // -----------------------------------------------------------------------
    group('TC-10: AskUserRequested', () {
      test('creates askUser message with options', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'AskUserRequested',
              'questions': [
                {
                  'text': 'Which framework?',
                  'options': [
                    {'label': 'React'},
                    {'label': 'Vue'},
                    {'label': 'Angular'},
                  ],
                },
              ],
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        final msg = messages[0];
        expect(msg.type, ChatMessageType.askUser);
        expect(msg.askUser, isNotNull);
        expect(msg.askUser!.question, 'Which framework?');
        expect(msg.askUser!.options, ['React', 'Vue', 'Angular']);
        expect(msg.askUser!.status, AskUserStatus.pending);
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-11: answerAskUser sends correct payload
    // -----------------------------------------------------------------------
    group('TC-11: answerAskUser', () {
      test('sends correct JSON payload to WebSocket sink', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Inject an ask-user message so the repo can find the question text.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'AskUserRequested',
              'questions': [
                {
                  'text': 'Continue?',
                  'options': [
                    {'label': 'Yes'},
                    {'label': 'No'},
                  ],
                },
              ],
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Grab the generated ask-user id.
        final msgs = await repo.getMessages('session');
        final askId = msgs.first.askUser!.id;

        await repo.answerAskUser('session', askId, 'Yes');

        expect(channel.sink.sent, hasLength(1));
        final sentJson =
            jsonDecode(channel.sink.sent.last) as Map<String, dynamic>;
        expect(sentJson['type'], 'ask_user_answer');
        expect(sentJson['answers'], isList);
        final answer =
            (sentJson['answers'] as List).first as Map<String, dynamic>;
        expect(answer['question'], 'Continue?');
        expect(answer['answer'], 'Yes');
        repo.dispose();
      });

      test('updates ask-user status to answered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Inject ask-user message.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'AskUserRequested',
              'questions': [
                {
                  'text': 'Pick one',
                  'options': [
                    {'label': 'A'},
                    {'label': 'B'},
                  ],
                },
              ],
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final askId = (await repo.getMessages('s')).first.askUser!.id;

        // Listen for the update emission.
        final updatedFuture = repo.watchMessages().first;
        await repo.answerAskUser('s', askId, 'B');

        final updated = await updatedFuture;
        final msg = updated.first;
        expect(msg.askUser!.status, AskUserStatus.answered);
        expect(msg.askUser!.answer, 'B');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-12: sendMessage sends correct WS payload
    // -----------------------------------------------------------------------
    group('TC-12: sendMessage', () {
      test('sends {type: user_message, text: ...} to sink', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        await repo.sendMessage('session', 'Hello agent');

        expect(channel.sink.sent, hasLength(1));
        final sentJson =
            jsonDecode(channel.sink.sent.last) as Map<String, dynamic>;
        expect(sentJson['type'], 'user_message');
        expect(sentJson['text'], 'Hello agent');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-14: Malformed JSON doesn't crash
    // -----------------------------------------------------------------------
    group('TC-14: malformed JSON', () {
      test('does not crash and repo stays functional', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Send garbage.
        channel.receive('not valid json {{{');
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Repo should still work -- send a valid event.
        final messagesFuture = repo.watchMessages().first;
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-after-garbage',
                'role': 'assistant',
                'content': 'Still alive',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );

        final messages = await messagesFuture;
        expect(messages, hasLength(1));
        expect(messages[0].content, 'Still alive');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-15: Multiple MessageCompleted for same agent message ID
    // -----------------------------------------------------------------------
    group('TC-15: duplicate agent MessageCompleted', () {
      test('same ID sent twice results in 1 message', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // First.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'agent-1',
                'role': 'assistant',
                'content': 'First version',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Second with same ID.
        final messagesFuture = repo.watchMessages().first;
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'agent-1',
                'role': 'assistant',
                'content': 'Updated version',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );

        final messages = await messagesFuture;
        expect(messages, hasLength(1));
        expect(messages[0].content, 'Updated version');
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-16: Timestamp parsing -- ISO string
    // -----------------------------------------------------------------------
    group('TC-16: ISO string timestamp', () {
      test('parses ISO 8601 timestamp correctly', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'init',
            'messages': [
              {
                'id': 'msg-iso',
                'role': 'assistant',
                'content': 'ISO timestamp',
                'timestamp': '2026-02-28T14:46:03.990Z',
              },
            ],
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].timestamp.year, 2026);
        expect(messages[0].timestamp.month, 2);
        expect(messages[0].timestamp.day, 28);
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-17: Timestamp parsing -- int milliseconds
    // -----------------------------------------------------------------------
    group('TC-17: int timestamp', () {
      test('parses int milliseconds timestamp correctly', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'init',
            'messages': [
              {
                'id': 'msg-int-ts',
                'role': 'assistant',
                'content': 'Int timestamp',
                'timestamp': _fixedTimestampMs,
              },
            ],
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].timestamp, DateTime(2026, 3, 1, 12, 0));
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // TC-18: Full conversation flow -- no duplicates
    // -----------------------------------------------------------------------
    group('TC-18: full conversation flow', () {
      test('complete flow produces exact correct message count', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Step 1: Init empty.
        channel.receive(
          jsonEncode({'type': 'init', 'messages': <dynamic>[]}),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Step 2: User sends message (optimistic).
        await repo.sendMessage('session', 'Explain this code');
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Verify: 1 optimistic user message.
        final afterSend = await repo.getMessages('session');
        expect(afterSend, hasLength(1));
        expect(afterSend[0].id, startsWith('user-'));
        expect(afterSend[0].content, 'Explain this code');

        // Step 3: Server echoes user message (different ID, same content).
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'server-user-uuid',
                'role': 'user',
                'content': 'Explain this code',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Verify: still 1 user message (dedup worked).
        final afterEcho = await repo.getMessages('session');
        expect(afterEcho, hasLength(1));
        expect(afterEcho[0].id, 'server-user-uuid');

        // Step 4: Agent response.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'agent-response-1',
                'role': 'assistant',
                'content': 'Let me look at the code...',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Step 5: Tool execution started.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionStarted',
              'execution': {
                'callId': 'tc-read-1',
                'toolName': 'read_file',
                'arguments': {'path': '/src/main.dart'},
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Step 6: Tool execution completed.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionCompleted',
              'execution': {
                'callId': 'tc-read-1',
                'result': '50 lines of code',
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Step 7: Agent's final answer.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'agent-final-1',
                'role': 'assistant',
                'content': 'This code does X, Y, Z.',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Final verification.
        final finalMessages = await repo.getMessages('session');

        // Expected: 1 user + 1 agent response + 1 tool call + 1 agent final
        // = 4 messages total. No duplicates.
        expect(finalMessages, hasLength(4));

        // User message (replaced by server version).
        expect(finalMessages[0].id, 'server-user-uuid');
        expect(finalMessages[0].type, ChatMessageType.userMessage);
        expect(finalMessages[0].content, 'Explain this code');

        // Agent initial response.
        expect(finalMessages[1].id, 'agent-response-1');
        expect(finalMessages[1].type, ChatMessageType.agentMessage);

        // Tool call (completed).
        expect(finalMessages[2].id, 'tc-read-1');
        expect(finalMessages[2].type, ChatMessageType.toolCall);
        expect(finalMessages[2].toolCall!.status, ToolCallStatus.completed);
        expect(finalMessages[2].toolCall!.result, '50 lines of code');

        // Agent final answer.
        expect(finalMessages[3].id, 'agent-final-1');
        expect(finalMessages[3].type, ChatMessageType.agentMessage);
        expect(finalMessages[3].content, 'This code does X, Y, Z.');

        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // Tool call dedup (MessageCompleted with tool data)
    // -----------------------------------------------------------------------
    group('tool call dedup', () {
      test(
        'TC-19: MessageCompleted with toolCall data does NOT create duplicate',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          final allEmissions = <List<ChatMessage>>[];
          repo.watchMessages().listen(allEmissions.add);

          // Step 1: ToolExecutionStarted creates tool call with status=running.
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'ToolExecutionStarted',
                'execution': {
                  'callId': 'call-xyz',
                  'toolName': 'read_file',
                  'arguments': {'path': '/src/main.dart'},
                },
              },
            }),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          final afterStart = await repo.getMessages('session');
          expect(afterStart, hasLength(1));
          expect(afterStart[0].id, 'call-xyz');
          expect(afterStart[0].toolCall!.status, ToolCallStatus.running);

          // Step 2: MessageCompleted with a tool message (different UUID).
          // This simulates the CLI publishing MessageCompleted after tool
          // finishes, with the Message entity's UUID (not the callId).
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'MessageCompleted',
                'message': {
                  'id': 'msg-uuid-different',
                  'role': 'tool',
                  'content': '',
                  'timestamp': _fixedTimestampIso,
                  'toolCall': {
                    'id': 'call-xyz',
                    'toolName': 'read_file',
                    'arguments': {'path': '/src/main.dart'},
                    'status': 'completed',
                    'result': '50 lines',
                  },
                },
              },
            }),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          // Should still be 1 message — the MessageCompleted with tool data
          // must be ignored (handled by ToolExecution events instead).
          final afterCompleted = await repo.getMessages('session');
          expect(afterCompleted, hasLength(1));
          expect(afterCompleted[0].id, 'call-xyz');

          repo.dispose();
        },
      );

      test(
        'TC-20: MessageCompleted with assistant+toolCall role is also skipped',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          // ToolExecutionStarted first.
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'ToolExecutionStarted',
                'execution': {
                  'callId': 'call-abc',
                  'toolName': 'search',
                  'arguments': {'query': 'test'},
                },
              },
            }),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          // MessageCompleted with role=assistant but hasToolCall.
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'MessageCompleted',
                'message': {
                  'id': 'assistant-msg-uuid',
                  'role': 'assistant',
                  'content': '',
                  'timestamp': _fixedTimestampIso,
                  'toolCall': {
                    'id': 'call-abc',
                    'toolName': 'search',
                    'arguments': {'query': 'test'},
                    'status': 'completed',
                  },
                },
              },
            }),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          final messages = await repo.getMessages('session');
          expect(messages, hasLength(1));
          expect(messages[0].id, 'call-abc');

          repo.dispose();
        },
      );
    });

    // -----------------------------------------------------------------------
    // Additional robustness tests
    // -----------------------------------------------------------------------
    group('robustness', () {
      test('binary data is decoded as UTF-8', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        // Send raw bytes instead of a string.
        final jsonString = jsonEncode({
          'type': 'event',
          'event': {
            'type': 'MessageCompleted',
            'message': {
              'id': 'msg-bin',
              'role': 'assistant',
              'content': 'From bytes',
              'timestamp': _fixedTimestampMs,
            },
          },
        });
        channel._incoming.add(utf8.encode(jsonString));

        final messages = await messagesFuture;
        expect(messages[0].content, 'From bytes');
        repo.dispose();
      });

      test('event with null event field is silently ignored', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        channel.receive(jsonEncode({'type': 'event'}));
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('session');
        expect(messages, isEmpty);
        repo.dispose();
      });

      test('init with no messages key results in empty list', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;
        channel.receive(jsonEncode({'type': 'init'}));

        final messages = await messagesFuture;
        expect(messages, isEmpty);
        repo.dispose();
      });

      test('unknown event type is silently ignored', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {'type': 'StreamingProgress', 'data': 'partial text'},
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('session');
        expect(messages, isEmpty);
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // getMessages
    // -----------------------------------------------------------------------
    group('getMessages', () {
      test('returns current messages as unmodifiable list', () async {
        final (:repo, :channel) = await _createConnectedRepo();

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
            ],
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('session');

        expect(messages, hasLength(1));
        expect(messages[0].content, 'Hello');
        // Verify unmodifiable.
        expect(
          () => messages.add(
            ChatMessage(
              id: 'x',
              type: ChatMessageType.userMessage,
              content: '',
              timestamp: DateTime.now(),
            ),
          ),
          throwsUnsupportedError,
        );
        repo.dispose();
      });

      test('returns empty list before any messages arrive', () async {
        final (:repo, channel: _) = await _createConnectedRepo();

        final messages = await repo.getMessages('session');
        expect(messages, isEmpty);
        repo.dispose();
      });
    });

    // -----------------------------------------------------------------------
    // Dispose
    // -----------------------------------------------------------------------
    group('dispose', () {
      test('closes sink on dispose', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        repo.dispose();

        expect(channel.sink.isClosed, isTrue);
      });
    });
  });
}

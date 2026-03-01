import 'dart:async';
import 'dart:convert';

import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// ---------------------------------------------------------------------------
// Fakes (same as ws_chat_repository_test.dart)
// ---------------------------------------------------------------------------

class _FakeWebSocketSink implements WebSocketSink {
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

class _FakeWebSocketChannel implements WebSocketChannel {
  _FakeWebSocketChannel()
    : _incoming = StreamController<dynamic>.broadcast(),
      sink = _FakeWebSocketSink();

  final StreamController<dynamic> _incoming;

  @override
  final _FakeWebSocketSink sink;

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

/// Creates a repository with a [_FakeWebSocketChannel] and connects it.
Future<({WsChatRepository repo, _FakeWebSocketChannel channel})>
_createConnectedRepo() async {
  final channel = _FakeWebSocketChannel();
  final repo = WsChatRepository(
    wsUrl: 'ws://localhost:8765',
    channelFactory: (_) => channel,
  );
  await repo.connect();
  return (repo: repo, channel: channel);
}

/// Fixed timestamp in milliseconds for tests that use int format.
final _fixedTimestampMs = DateTime(2026, 3, 1, 12, 0).millisecondsSinceEpoch;

/// Fixed ISO 8601 timestamp string matching CLI output format.
const _fixedTimestampIso = '2026-02-28T14:46:03.990Z';

// ---------------------------------------------------------------------------
// Tests: Real CLI event flow
// ---------------------------------------------------------------------------

void main() {
  group('WsChatRepository - real CLI message flow', () {
    // TC-WS-01
    group('TC-WS-01: init with empty messages', () {
      test('emits empty list through watchMessages', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'init',
            'messages': <dynamic>[],
            'meta': {
              'projectName': 'test',
              'projectPath': '/tmp',
              'sessionId': 's1',
            },
          }),
        );

        final messages = await messagesFuture;
        expect(messages, isEmpty);
        repo.dispose();
      });
    });

    // TC-WS-02
    group('TC-WS-02: init with messages', () {
      test('parses init messages and emits through watchMessages', () async {
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
                'content': 'Hi there!',
                'timestamp': _fixedTimestampMs,
              },
            ],
            'meta': {
              'projectName': 'test',
              'projectPath': '/tmp',
              'sessionId': 's1',
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(2));
        expect(messages[0].id, 'msg-1');
        expect(messages[0].type, ChatMessageType.userMessage);
        expect(messages[0].content, 'Hello');
        expect(messages[1].id, 'msg-2');
        expect(messages[1].type, ChatMessageType.agentMessage);
        expect(messages[1].content, 'Hi there!');
        repo.dispose();
      });
    });

    // TC-WS-03
    group('TC-WS-03: MessageCompleted (assistant) via event wrapper', () {
      test(
        'assistant message appears in watchMessages with correct content and type',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          final messagesFuture = repo.watchMessages().first;

          // This is the exact format the CLI sends via MobileProxyServer.
          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'MessageCompleted',
                'message': {
                  'id': 'msg-1',
                  'role': 'assistant',
                  'content': 'Привет!',
                  'timestamp': _fixedTimestampIso,
                  'isStreaming': false,
                  'isComplete': true,
                },
              },
            }),
          );

          final messages = await messagesFuture;

          expect(messages, hasLength(1));
          expect(messages[0].id, 'msg-1');
          expect(messages[0].type, ChatMessageType.agentMessage);
          expect(messages[0].content, 'Привет!');
          repo.dispose();
        },
      );
    });

    // TC-WS-04
    group('TC-WS-04: MessageCompleted (user) via event wrapper', () {
      test('user message from event has correct type', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-user-1',
                'role': 'user',
                'content': 'My question',
                'timestamp': _fixedTimestampIso,
                'isStreaming': false,
                'isComplete': true,
              },
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].id, 'msg-user-1');
        expect(messages[0].type, ChatMessageType.userMessage);
        expect(messages[0].content, 'My question');
        repo.dispose();
      });
    });

    // TC-WS-05
    group('TC-WS-05: ToolExecutionStarted via event wrapper', () {
      test('creates tool call message with running status', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'ToolExecutionStarted',
              'execution': {
                'callId': 'tc-1',
                'toolName': 'read_file',
                'arguments': {'path': '/src/main.dart'},
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
        expect(msg.toolCall!.toolName, 'read_file');
        expect(msg.toolCall!.arguments, {'path': '/src/main.dart'});
        expect(msg.toolCall!.status, ToolCallStatus.running);
        repo.dispose();
      });
    });

    // TC-WS-06
    group('TC-WS-06: ISO 8601 timestamp string parsing', () {
      test('ISO string timestamp parses correctly (not crash)', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-iso',
                'role': 'assistant',
                'content': 'Response with ISO timestamp',
                'timestamp': '2026-02-28T14:46:03.990Z',
                'isStreaming': false,
                'isComplete': true,
              },
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].content, 'Response with ISO timestamp');
        // Verify that the ISO timestamp was actually parsed, not just
        // defaulting to DateTime.now().
        expect(messages[0].timestamp.year, 2026);
        expect(messages[0].timestamp.month, 2);
        expect(messages[0].timestamp.day, 28);
        repo.dispose();
      });

      test('int timestamp still works (backwards compatibility)', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-int-ts',
                'role': 'assistant',
                'content': 'With int timestamp',
                'timestamp': _fixedTimestampMs,
              },
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(1));
        expect(messages[0].content, 'With int timestamp');
        expect(messages[0].timestamp, DateTime(2026, 3, 1, 12, 0));
        repo.dispose();
      });

      test('null timestamp falls back to DateTime.now()', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final before = DateTime.now();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-no-ts',
                'role': 'assistant',
                'content': 'No timestamp',
              },
            },
          }),
        );

        final messages = await messagesFuture;
        final after = DateTime.now();

        expect(messages, hasLength(1));
        final ts = messages[0].timestamp;
        expect(
          ts.isAfter(before) || ts.isAtSameMomentAs(before),
          isTrue,
        );
        expect(
          ts.isBefore(after) || ts.isAtSameMomentAs(after),
          isTrue,
        );
        repo.dispose();
      });
    });

    // TC-WS-07
    group(
      'TC-WS-07: full flow init -> user message -> assistant response',
      () {
        test(
          'user optimistic update + assistant event both appear in stream',
          () async {
            final (:repo, :channel) = await _createConnectedRepo();

            // Step 1: init with empty messages.
            channel.receive(
              jsonEncode({
                'type': 'init',
                'messages': <dynamic>[],
                'meta': {
                  'projectName': 'test',
                  'projectPath': '/tmp',
                  'sessionId': 's1',
                },
              }),
            );
            await Future<void>.delayed(const Duration(milliseconds: 50));

            // Step 2: user sends message (optimistic update).
            await repo.sendMessage('s1', 'Hello agent');
            await Future<void>.delayed(const Duration(milliseconds: 50));

            // Verify user message is present.
            final afterUser = await repo.getMessages('s1');
            expect(afterUser, hasLength(1));
            expect(afterUser[0].type, ChatMessageType.userMessage);
            expect(afterUser[0].content, 'Hello agent');

            // Step 3: agent responds via WS event.
            final messagesFuture = repo.watchMessages().first;

            channel.receive(
              jsonEncode({
                'type': 'event',
                'event': {
                  'type': 'MessageCompleted',
                  'message': {
                    'id': 'msg-assistant-1',
                    'role': 'assistant',
                    'content': 'Hello! How can I help?',
                    'timestamp': _fixedTimestampIso,
                    'isStreaming': false,
                    'isComplete': true,
                  },
                },
              }),
            );

            final messages = await messagesFuture;

            // Both messages should be present.
            expect(messages, hasLength(2));

            // First: user message (optimistic).
            expect(messages[0].type, ChatMessageType.userMessage);
            expect(messages[0].content, 'Hello agent');

            // Second: assistant response from event.
            expect(messages[1].id, 'msg-assistant-1');
            expect(messages[1].type, ChatMessageType.agentMessage);
            expect(messages[1].content, 'Hello! How can I help?');
            repo.dispose();
          },
        );
      },
    );

    // Additional edge case: init messages with ISO timestamps (as sent by CLI)
    group('init with ISO timestamps from CLI', () {
      test('parses init messages that have ISO string timestamps', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        // The CLI serializes Date objects via JSON.stringify which produces
        // ISO 8601 strings -- not int milliseconds.
        channel.receive(
          jsonEncode({
            'type': 'init',
            'messages': [
              {
                'id': 'msg-1',
                'role': 'user',
                'content': 'Hello',
                'timestamp': '2026-02-28T10:00:00.000Z',
                'isStreaming': false,
                'isComplete': true,
              },
              {
                'id': 'msg-2',
                'role': 'assistant',
                'content': 'World',
                'timestamp': '2026-02-28T10:00:01.000Z',
                'isStreaming': false,
                'isComplete': true,
              },
            ],
            'meta': {
              'projectName': 'test',
              'projectPath': '/tmp',
              'sessionId': 's1',
            },
          }),
        );

        final messages = await messagesFuture;

        expect(messages, hasLength(2));
        expect(messages[0].content, 'Hello');
        expect(messages[0].type, ChatMessageType.userMessage);
        expect(messages[0].timestamp.year, 2026);
        expect(messages[1].content, 'World');
        expect(messages[1].type, ChatMessageType.agentMessage);
        expect(messages[1].timestamp.year, 2026);
        repo.dispose();
      });
    });

    // TC-WS-CLI-FILTER: CLI-only messages are filtered out
    group('TC-WS-CLI-FILTER: CLI-only visual elements are filtered', () {
      test('separator line "─── Supervisor ───" is filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        // Send a real assistant message first so we have something to compare.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-real',
                'role': 'assistant',
                'content': 'Real response',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // Now send a separator line -- should be silently dropped.
        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-sep',
                'role': 'assistant',
                'content': '─── Supervisor ───',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('s1');
        expect(messages, hasLength(1));
        expect(messages[0].content, 'Real response');
        repo.dispose();
      });

      test(
        'agent separator "─── Code Agent [abc]: Task ───" is filtered',
        () async {
          final (:repo, :channel) = await _createConnectedRepo();

          channel.receive(
            jsonEncode({
              'type': 'event',
              'event': {
                'type': 'MessageCompleted',
                'message': {
                  'id': 'msg-agent-sep',
                  'role': 'assistant',
                  'content': '─── Code Agent [abc123]: Implement feature ───',
                  'timestamp': _fixedTimestampIso,
                },
              },
            }),
          );
          await Future<void>.delayed(const Duration(milliseconds: 50));

          final messages = await repo.getMessages('s1');
          expect(messages, isEmpty);
          repo.dispose();
        },
      );

      test('lifecycle "+ Agent spawned" is filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-spawned',
                'role': 'assistant',
                'content': '+ Code Agent [abc] spawned: "Task"',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('s1');
        expect(messages, isEmpty);
        repo.dispose();
      });

      test('lifecycle "\u2713 Agent completed" is filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-completed',
                'role': 'assistant',
                'content': '\u2713 Code Agent completed: "Task"',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('s1');
        expect(messages, isEmpty);
        repo.dispose();
      });

      test('lifecycle "\u2717 Agent failed" is filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-failed',
                'role': 'assistant',
                'content': '\u2717 Agent failed',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('s1');
        expect(messages, isEmpty);
        repo.dispose();
      });

      test('lifecycle "\u21BB Agent retrying" is filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-retry',
                'role': 'assistant',
                'content': '\u21BB Agent retrying',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 50));

        final messages = await repo.getMessages('s1');
        expect(messages, isEmpty);
        repo.dispose();
      });

      test('normal assistant message is NOT filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-normal',
                'role': 'assistant',
                'content': 'Here is the analysis of your code.',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );

        final messages = await messagesFuture;
        expect(messages, hasLength(1));
        expect(messages[0].content, 'Here is the analysis of your code.');
        repo.dispose();
      });

      test('user message starting with "+ " is NOT filtered', () async {
        final (:repo, :channel) = await _createConnectedRepo();

        final messagesFuture = repo.watchMessages().first;

        channel.receive(
          jsonEncode({
            'type': 'event',
            'event': {
              'type': 'MessageCompleted',
              'message': {
                'id': 'msg-user-plus',
                'role': 'user',
                'content': '+ some user text',
                'timestamp': _fixedTimestampIso,
              },
            },
          }),
        );

        final messages = await messagesFuture;
        expect(messages, hasLength(1));
        expect(messages[0].type, ChatMessageType.userMessage);
        expect(messages[0].content, '+ some user text');
        repo.dispose();
      });
    });
  });
}

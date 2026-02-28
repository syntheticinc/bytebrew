import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/plan.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/message_mapper.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MessageMapper.fromSnapshot', () {
    final fixedTimestampMs = DateTime(
      2026,
      1,
      15,
      10,
      30,
    ).millisecondsSinceEpoch;

    group('agent message', () {
      test('parses role assistant as agentMessage', () {
        final snapshot = {
          'id': 'msg-1',
          'role': 'assistant',
          'content': 'Hello world',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.id, 'msg-1');
        expect(msg.type, ChatMessageType.agentMessage);
        expect(msg.content, 'Hello world');
        expect(msg.timestamp, DateTime(2026, 1, 15, 10, 30));
        expect(msg.toolCall, isNull);
        expect(msg.plan, isNull);
        expect(msg.askUser, isNull);
      });
    });

    group('user message', () {
      test('parses role user as userMessage', () {
        final snapshot = {
          'id': 'msg-2',
          'role': 'user',
          'content': 'What is this?',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.userMessage);
        expect(msg.content, 'What is this?');
      });
    });

    group('system message', () {
      test('parses role system as systemMessage', () {
        final snapshot = {
          'id': 'msg-sys',
          'role': 'system',
          'content': 'System prompt',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.systemMessage);
      });

      test('defaults to systemMessage for unknown role', () {
        final snapshot = {
          'id': 'msg-unknown',
          'role': 'unknown_role',
          'content': '',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.systemMessage);
      });
    });

    group('tool call', () {
      test('parses assistant with toolCall as toolCall type', () {
        final snapshot = {
          'id': 'msg-tc',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolCall': {
            'id': 'tc-1',
            'toolName': 'read_file',
            'arguments': {'path': '/src/main.dart'},
            'status': 'completed',
            'result': '50 lines',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.toolCall);
        expect(msg.toolCall, isNotNull);
        expect(msg.toolCall!.id, 'tc-1');
        expect(msg.toolCall!.toolName, 'read_file');
        expect(msg.toolCall!.arguments, {'path': '/src/main.dart'});
        expect(msg.toolCall!.status, ToolCallStatus.completed);
        expect(msg.toolCall!.result, '50 lines');
      });

      test('parses tool role with toolCall as toolCall type', () {
        final snapshot = {
          'id': 'msg-tc2',
          'role': 'tool',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolCall': {
            'id': 'tc-2',
            'name': 'search_code',
            'arguments': {'query': 'class'},
            'status': 'running',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.toolCall);
        expect(msg.toolCall!.toolName, 'search_code');
        expect(msg.toolCall!.status, ToolCallStatus.running);
      });

      test('parses tool role without toolCall as toolResult', () {
        final snapshot = {
          'id': 'msg-tr',
          'role': 'tool',
          'content': 'tool output here',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.toolResult);
      });

      test('parses tool role with toolResult as toolResult', () {
        final snapshot = {
          'id': 'msg-tr2',
          'role': 'tool',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolResult': {'output': 'some result'},
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.toolResult);
      });
    });

    group('tool call status mapping', () {
      test('maps completed status', () {
        final snapshot = _toolCallSnapshot(status: 'completed');
        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.status, ToolCallStatus.completed);
      });

      test('maps failed status', () {
        final snapshot = _toolCallSnapshot(status: 'failed');
        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.status, ToolCallStatus.failed);
      });

      test('maps running status', () {
        final snapshot = _toolCallSnapshot(status: 'running');
        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.status, ToolCallStatus.running);
      });

      test('defaults to running for unknown status', () {
        final snapshot = _toolCallSnapshot(status: 'something_else');
        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.status, ToolCallStatus.running);
      });

      test('defaults to running for null status', () {
        final snapshot = _toolCallSnapshot(status: null);
        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.status, ToolCallStatus.running);
      });
    });

    group('tool call fields', () {
      test('parses fullResult and error fields', () {
        final snapshot = {
          'id': 'msg-tc-err',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolCall': {
            'id': 'tc-err',
            'toolName': 'execute',
            'arguments': <String, dynamic>{},
            'status': 'failed',
            'result': 'short result',
            'fullResult': 'very long full result text',
            'error': 'command failed with exit code 1',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.fullResult, 'very long full result text');
        expect(msg.toolCall!.error, 'command failed with exit code 1');
        expect(msg.toolCall!.result, 'short result');
      });

      test('falls back to name when toolName is absent', () {
        final snapshot = {
          'id': 'msg-tc-name',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolCall': {
            'id': 'tc-name',
            'name': 'fallback_name',
            'arguments': <String, dynamic>{},
            'status': 'running',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.toolName, 'fallback_name');
      });

      test('converts non-string argument values to strings', () {
        final snapshot = {
          'id': 'msg-tc-args',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolCall': {
            'id': 'tc-args',
            'toolName': 'test',
            'arguments': {'count': 42, 'enabled': true},
            'status': 'running',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.toolCall!.arguments, {'count': '42', 'enabled': 'true'});
      });
    });

    group('plan update', () {
      test('parses plan with steps', () {
        final snapshot = {
          'id': 'msg-plan',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {
            'goal': 'Refactor the module',
            'steps': [
              {'description': 'Analyze code', 'status': 'completed'},
              {'description': 'Extract interface', 'status': 'in_progress'},
              {'description': 'Write tests', 'status': 'pending'},
            ],
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.planUpdate);
        expect(msg.plan, isNotNull);
        expect(msg.plan!.goal, 'Refactor the module');
        expect(msg.plan!.steps, hasLength(3));

        expect(msg.plan!.steps[0].index, 0);
        expect(msg.plan!.steps[0].description, 'Analyze code');
        expect(msg.plan!.steps[0].status, PlanStepStatus.completed);

        expect(msg.plan!.steps[1].index, 1);
        expect(msg.plan!.steps[1].description, 'Extract interface');
        expect(msg.plan!.steps[1].status, PlanStepStatus.inProgress);

        expect(msg.plan!.steps[2].index, 2);
        expect(msg.plan!.steps[2].description, 'Write tests');
        expect(msg.plan!.steps[2].status, PlanStepStatus.pending);
      });

      test('accepts inProgress as step status variant', () {
        final snapshot = {
          'id': 'msg-plan2',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {
            'goal': 'Test',
            'steps': [
              {'description': 'Step 1', 'status': 'inProgress'},
            ],
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.plan!.steps[0].status, PlanStepStatus.inProgress);
      });

      test('defaults step status to pending for unknown values', () {
        final snapshot = {
          'id': 'msg-plan3',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {
            'goal': 'Test',
            'steps': [
              {'description': 'Step 1', 'status': 'unknown_status'},
            ],
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.plan!.steps[0].status, PlanStepStatus.pending);
      });

      test('parses step completedAt datetime', () {
        final snapshot = {
          'id': 'msg-plan4',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {
            'goal': 'Test',
            'steps': [
              {
                'description': 'Done step',
                'status': 'completed',
                'completedAt': '2026-01-15T10:30:00.000',
              },
            ],
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.plan!.steps[0].completedAt, isNotNull);
        expect(msg.plan!.steps[0].completedAt!.year, 2026);
      });

      test('handles empty steps list', () {
        final snapshot = {
          'id': 'msg-plan5',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {'goal': 'Empty plan', 'steps': <dynamic>[]},
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.planUpdate);
        expect(msg.plan!.steps, isEmpty);
      });

      test('handles missing steps key', () {
        final snapshot = {
          'id': 'msg-plan6',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {'goal': 'No steps key'},
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.plan!.steps, isEmpty);
      });
    });

    group('ask user', () {
      test('parses askUser with options', () {
        final snapshot = {
          'id': 'msg-ask',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'askUser': {
            'id': 'ask-1',
            'question': 'Which framework?',
            'options': ['React', 'Vue', 'Angular'],
            'status': 'pending',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.askUser);
        expect(msg.askUser, isNotNull);
        expect(msg.askUser!.id, 'ask-1');
        expect(msg.askUser!.question, 'Which framework?');
        expect(msg.askUser!.options, ['React', 'Vue', 'Angular']);
        expect(msg.askUser!.status, AskUserStatus.pending);
        expect(msg.askUser!.answer, isNull);
      });

      test('parses answered askUser with answer', () {
        final snapshot = {
          'id': 'msg-ask2',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'askUser': {
            'id': 'ask-2',
            'question': 'Continue?',
            'options': ['Yes', 'No'],
            'status': 'answered',
            'answer': 'Yes',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.askUser!.status, AskUserStatus.answered);
        expect(msg.askUser!.answer, 'Yes');
      });

      test('defaults askUser status to pending for unknown values', () {
        final snapshot = {
          'id': 'msg-ask3',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'askUser': {
            'id': 'ask-3',
            'question': 'Test?',
            'options': <dynamic>[],
            'status': 'unknown',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.askUser!.status, AskUserStatus.pending);
      });

      test('handles empty options list', () {
        final snapshot = {
          'id': 'msg-ask4',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'askUser': {
            'id': 'ask-4',
            'question': 'Free text input?',
            'options': <dynamic>[],
            'status': 'pending',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.askUser!.options, isEmpty);
      });
    });

    group('reasoning', () {
      test('parses assistant with reasoning field as reasoning type', () {
        final snapshot = {
          'id': 'msg-reason',
          'role': 'assistant',
          'content': 'Thinking about this...',
          'timestamp': fixedTimestampMs,
          'reasoning': 'Let me analyze the problem',
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.reasoning);
      });
    });

    group('type resolution priority', () {
      test('askUser takes priority over plan', () {
        final snapshot = {
          'id': 'msg-priority1',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'askUser': {
            'id': 'a1',
            'question': 'Q?',
            'options': <dynamic>[],
            'status': 'pending',
          },
          'plan': {'goal': 'G', 'steps': <dynamic>[]},
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.askUser);
      });

      test('plan takes priority over toolCall for non-tool roles', () {
        final snapshot = {
          'id': 'msg-priority2',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'plan': {'goal': 'G', 'steps': <dynamic>[]},
          'toolCall': {
            'id': 'tc',
            'toolName': 't',
            'arguments': <String, dynamic>{},
            'status': 'running',
          },
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.planUpdate);
      });

      test('toolCall takes priority over reasoning for assistant role', () {
        final snapshot = {
          'id': 'msg-priority3',
          'role': 'assistant',
          'content': '',
          'timestamp': fixedTimestampMs,
          'toolCall': {
            'id': 'tc',
            'toolName': 't',
            'arguments': <String, dynamic>{},
            'status': 'running',
          },
          'reasoning': 'thinking...',
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.toolCall);
      });
    });

    group('null and missing fields', () {
      test('handles null content gracefully', () {
        final snapshot = {
          'id': 'msg-null',
          'role': 'assistant',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.content, '');
        expect(msg.type, ChatMessageType.agentMessage);
      });

      test('handles missing id with empty string default', () {
        final snapshot = {
          'role': 'user',
          'content': 'Hello',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.id, '');
        expect(msg.type, ChatMessageType.userMessage);
      });

      test('handles missing role with empty string default', () {
        final snapshot = {
          'id': 'msg-norole',
          'content': 'No role',
          'timestamp': fixedTimestampMs,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.type, ChatMessageType.systemMessage);
      });

      test('uses DateTime.now when timestamp is missing', () {
        final before = DateTime.now();
        final snapshot = {
          'id': 'msg-notime',
          'role': 'user',
          'content': 'No timestamp',
        };

        final msg = MessageMapper.fromSnapshot(snapshot);
        final after = DateTime.now();

        expect(
          msg.timestamp.isAfter(before) || msg.timestamp == before,
          isTrue,
        );
        expect(msg.timestamp.isBefore(after) || msg.timestamp == after, isTrue);
      });
    });

    group('timestamp parsing', () {
      test('converts milliseconds to DateTime correctly', () {
        final dt = DateTime(2026, 6, 15, 14, 30, 45);
        final snapshot = {
          'id': 'msg-ts',
          'role': 'assistant',
          'content': '',
          'timestamp': dt.millisecondsSinceEpoch,
        };

        final msg = MessageMapper.fromSnapshot(snapshot);

        expect(msg.timestamp.year, 2026);
        expect(msg.timestamp.month, 6);
        expect(msg.timestamp.day, 15);
        expect(msg.timestamp.hour, 14);
        expect(msg.timestamp.minute, 30);
      });
    });
  });
}

/// Helper to build a minimal tool call snapshot with a given status.
Map<String, dynamic> _toolCallSnapshot({required String? status}) {
  return {
    'id': 'msg-tc-status',
    'role': 'assistant',
    'content': '',
    'timestamp': DateTime(2026, 1, 1).millisecondsSinceEpoch,
    'toolCall': {
      'id': 'tc-status',
      'toolName': 'test_tool',
      'arguments': <String, dynamic>{},
      'status': status,
    },
  };
}

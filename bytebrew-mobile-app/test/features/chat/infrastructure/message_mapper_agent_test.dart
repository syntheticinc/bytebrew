import 'package:bytebrew_mobile/features/chat/infrastructure/message_mapper.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MessageMapper agentId', () {
    test('parses agentId from snapshot', () {
      final msg = MessageMapper.fromSnapshot({
        'id': 'msg-1',
        'role': 'assistant',
        'content': 'Hello',
        'agentId': 'agent-code-1',
      });
      expect(msg.agentId, 'agent-code-1');
    });

    test('agentId is null when not present', () {
      final msg = MessageMapper.fromSnapshot({
        'id': 'msg-2',
        'role': 'assistant',
        'content': 'Hello',
      });
      expect(msg.agentId, isNull);
    });

    test('tool call snapshot preserves agentId', () {
      final msg = MessageMapper.fromSnapshot({
        'id': 'msg-3',
        'role': 'tool',
        'content': '',
        'agentId': 'agent-code-2',
        'toolCall': {
          'id': 'tc-1',
          'toolName': 'read_file',
          'arguments': {'path': '/tmp'},
          'status': 'completed',
        },
      });
      expect(msg.agentId, 'agent-code-2');
      expect(msg.toolCall, isNotNull);
      expect(msg.toolCall!.toolName, 'read_file');
    });

    test('agentId is null for user messages without it', () {
      final msg = MessageMapper.fromSnapshot({
        'id': 'msg-4',
        'role': 'user',
        'content': 'Hey',
      });
      expect(msg.agentId, isNull);
    });

    test('system message preserves agentId', () {
      final msg = MessageMapper.fromSnapshot({
        'id': 'msg-5',
        'role': 'system',
        'content': 'Agent started: coder',
        'agentId': 'agent-coder',
      });
      expect(msg.agentId, 'agent-coder');
    });
  });
}

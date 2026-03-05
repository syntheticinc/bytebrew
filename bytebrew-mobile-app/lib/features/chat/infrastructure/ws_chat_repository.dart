import 'dart:async';

import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/ask_user.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/event_parser.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/message_mapper.dart';

/// [ChatRepository] backed by a WebSocket connection to the CLI.
///
/// Receives initial messages from the `init` payload and real-time events
/// from the CLI event stream.
class WsChatRepository implements ChatRepository {
  WsChatRepository({required WsConnection connection})
    : _connection = connection;

  final WsConnection _connection;
  final List<ChatMessage> _messages = [];
  final Map<String, AgentInfo> _agents = {};
  final _messagesController = StreamController<List<ChatMessage>>.broadcast();
  final _agentsController = StreamController<List<AgentInfo>>.broadcast();
  StreamSubscription<Map<String, dynamic>>? _eventSub;

  /// Loads initial messages from the connection and subscribes to events.
  void loadInitMessages() {
    for (final json in _connection.initMessages) {
      final msg = MessageMapper.fromSnapshot(json);
      _upsertMessage(msg);
    }
    _emitMessages();

    // Subscribe to real-time events.
    _eventSub = _connection.events.listen(_handleEvent);
  }

  void _handleEvent(Map<String, dynamic> event) {
    final type = event['type'] as String?;

    // Agent lifecycle -> update agents, not messages.
    final agentInfo = parseAgentLifecycle(event);
    if (agentInfo != null) {
      _agents[agentInfo.agentId] = agentInfo;
      _emitAgents();
      return;
    }

    // Tool execution completed -> update existing tool call by callId.
    if (type == 'ToolExecutionCompleted') {
      final exec = event['execution'] as Map<String, dynamic>?;
      if (exec != null) {
        final callId = exec['callId'] as String? ?? '';
        final index = _messages.indexWhere((m) => m.id == callId);
        if (index != -1) {
          final original = _messages[index];
          if (original.toolCall != null) {
            _messages[index] = original.copyWith(
              type: ChatMessageType.toolResult,
              toolCall: original.toolCall!.copyWith(
                status: exec['error'] != null
                    ? ToolCallStatus.failed
                    : ToolCallStatus.completed,
                result: exec['summary'] as String? ?? exec['result'] as String?,
                error: exec['error'] as String?,
              ),
            );
            _emitMessages();
            return;
          }
        }
      }
    }

    final msg = parseEventToChatMessage(event);
    if (msg != null) {
      _upsertMessage(msg);
      _emitMessages();
    }
  }

  @override
  Future<List<ChatMessage>> getMessages(String sessionId) async =>
      List.unmodifiable(_messages);

  @override
  Future<void> sendMessage(String sessionId, String text) async {
    // Optimistic update: add the user message immediately.
    final msg = ChatMessage(
      id: 'user-${DateTime.now().millisecondsSinceEpoch}',
      type: ChatMessageType.userMessage,
      content: text,
      timestamp: DateTime.now(),
    );
    _upsertMessage(msg);
    _emitMessages();
    _connection.sendUserMessage(text);
  }

  @override
  Future<void> answerAskUser(
    String sessionId,
    String askUserId,
    String answer,
  ) async {
    final question =
        _messages
            .where((m) => m.askUser?.id == askUserId)
            .firstOrNull
            ?.askUser
            ?.question ??
        '';

    _updateAskUserStatus(askUserId, answer);
    _emitMessages();
    _connection.sendAskUserAnswer(question, answer);
  }

  @override
  Future<void> cancel(String sessionId) async {
    _connection.sendCancel();
  }

  @override
  Stream<List<ChatMessage>> watchMessages() => _messagesController.stream;

  @override
  Stream<List<AgentInfo>> watchAgents() => _agentsController.stream;

  /// Releases resources. Call when the repository is no longer needed.
  void dispose() {
    _eventSub?.cancel();
    _messagesController.close();
    _agentsController.close();
  }

  void _upsertMessage(ChatMessage msg) {
    final index = _messages.indexWhere((m) => m.id == msg.id);
    if (index != -1) {
      _messages[index] = msg;
    } else {
      _messages.add(msg);
    }
  }

  void _updateAskUserStatus(String askUserId, String answer) {
    final index = _messages.indexWhere((m) => m.askUser?.id == askUserId);
    if (index == -1) return;
    final original = _messages[index];
    _messages[index] = original.copyWith(
      askUser: original.askUser!.copyWith(
        status: AskUserStatus.answered,
        answer: answer,
      ),
    );
  }

  void _emitMessages() {
    if (_messagesController.isClosed) return;
    _messages.sort((a, b) => a.timestamp.compareTo(b.timestamp));
    _messagesController.add(List.unmodifiable(_messages));
  }

  void _emitAgents() {
    if (_agentsController.isClosed) return;
    _agentsController.add(List.unmodifiable(_agents.values.toList()));
  }
}

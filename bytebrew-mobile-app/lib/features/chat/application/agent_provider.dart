import 'dart:async';

import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'agent_provider.g.dart';

/// Tracks agents from lifecycle events.
@riverpod
class Agents extends _$Agents {
  StreamSubscription<List<AgentInfo>>? _subscription;

  @override
  List<AgentInfo> build(String sessionId) {
    final repo = ref.watch(sessionChatRepositoryProvider(sessionId));

    _subscription?.cancel();
    final agentStream = repo.watchAgents();
    if (agentStream != null) {
      _subscription = agentStream.listen((agents) {
        state = agents;
      });
      ref.onDispose(() => _subscription?.cancel());
    }
    return [];
  }
}

/// Selected agent ID. null = supervisor view.
@riverpod
class SelectedAgent extends _$SelectedAgent {
  @override
  String? build(String sessionId) => null;

  void select(String? agentId) => state = agentId;
}

/// Filtered messages by selected agent.
@riverpod
List<ChatMessage> filteredChatMessages(Ref ref, String sessionId) {
  final messagesAsync = ref.watch(chatMessagesProvider(sessionId));
  final selectedAgentId = ref.watch(selectedAgentProvider(sessionId));

  final messages = messagesAsync.whenOrNull(data: (d) => d) ?? [];

  // No selected agent = supervisor view
  if (selectedAgentId == null) {
    return messages.where((m) {
      if (m.type == ChatMessageType.userMessage) return true;
      if (m.agentId == null || m.agentId == 'supervisor') return true;
      // Show lifecycle system messages for all agents
      if (m.type == ChatMessageType.systemMessage) return true;
      return false;
    }).toList();
  }

  // Agent view: only messages from this agent
  return messages.where((m) => m.agentId == selectedAgentId).toList();
}

/// Whether multi-agent mode is active.
@riverpod
bool isMultiAgent(Ref ref, String sessionId) {
  return ref.watch(agentsProvider(sessionId)).isNotEmpty;
}

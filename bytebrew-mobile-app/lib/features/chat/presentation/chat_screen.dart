import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/domain/chat_message.dart';
import '../../../core/domain/session.dart';
import '../../../core/infrastructure/ws/ws_connection.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/widgets/status_indicator.dart';
import '../../sessions/application/sessions_provider.dart';
import '../application/agent_provider.dart';
import '../application/chat_provider.dart';
import 'widgets/agent_lifecycle_widget.dart';
import 'widgets/agent_message_bubble.dart';
import 'widgets/agent_selector_bar.dart';
import 'widgets/ask_user_widget.dart';
import 'widgets/chat_input_bar.dart';
import 'widgets/connection_info_badge.dart';
import 'widgets/plan_widget.dart';
import 'widgets/reasoning_widget.dart';
import 'widgets/tool_call_widget.dart';
import 'widgets/user_message_bubble.dart';

/// Full chat screen showing messages, tool calls, plans, and input bar.
class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key, required this.sessionId});

  final String sessionId;

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  @override
  Widget build(BuildContext context) {
    final session = ref.watch(sessionByIdProvider(widget.sessionId));
    final messagesAsync = ref.watch(chatMessagesProvider(widget.sessionId));
    final isMultiAgent = ref.watch(isMultiAgentProvider(widget.sessionId));
    final selectedAgentId = isMultiAgent
        ? ref.watch(selectedAgentProvider(widget.sessionId))
        : null;
    final isAgentView = isMultiAgent && selectedAgentId != null;

    return Scaffold(
      appBar: _buildAppBar(context, session),
      body: Column(
        children: [
          if (isMultiAgent) AgentSelectorBar(sessionId: widget.sessionId),
          Expanded(child: _buildMessageList(messagesAsync, isMultiAgent)),
          SafeArea(
            child: ChatInputBar(
              sessionId: widget.sessionId,
              enabled: !isAgentView,
            ),
          ),
        ],
      ),
    );
  }

  PreferredSizeWidget _buildAppBar(BuildContext context, Session? session) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final connectionBadge = _buildConnectionBadge();

    return AppBar(
      title: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Flexible(
                child: Text(
                  session?.projectName ?? 'Chat',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (session != null) ...[
                const SizedBox(width: 4),
                StatusIndicator(color: AppColors.statusColor(session.status)),
              ],
            ],
          ),
          if (session != null)
            Text(
              session.currentTask ?? session.serverName,
              style: theme.textTheme.bodySmall?.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
        ],
      ),
      actions: [
        if (connectionBadge != null)
          Padding(
            padding: const EdgeInsets.only(right: 8),
            child: connectionBadge,
          ),
      ],
    );
  }

  /// Builds a connection info badge showing WS connection status.
  Widget? _buildConnectionBadge() {
    final status = ref.watch(wsConnectionProvider);
    return ConnectionInfoBadge(status: status);
  }

  Widget _buildMessageList(
    AsyncValue<List<ChatMessage>> messagesAsync,
    bool isMultiAgent,
  ) {
    return messagesAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, _) => _buildErrorState(error.toString()),
      data: (messages) {
        if (messages.isEmpty) {
          return _buildEmptyState();
        }

        // In multi-agent mode, use the filtered provider
        if (isMultiAgent) {
          return _buildFilteredMessageList();
        }

        return _buildMessageListView(messages);
      },
    );
  }

  Widget _buildFilteredMessageList() {
    final filtered = ref.watch(filteredChatMessagesProvider(widget.sessionId));
    if (filtered.isEmpty) {
      return _buildEmptyState();
    }
    return _buildMessageListView(filtered);
  }

  Widget _buildMessageListView(List<ChatMessage> messages) {
    return ListView.builder(
      reverse: true,
      padding: const EdgeInsets.symmetric(vertical: 8),
      itemCount: messages.length,
      itemBuilder: (context, index) {
        final message = messages[messages.length - 1 - index];
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 2),
          child: _buildMessageWidget(message),
        );
      },
    );
  }

  Widget _buildEmptyState() {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            'bb',
            style: theme.textTheme.displayLarge?.copyWith(
              color: AppColors.accent,
              fontWeight: FontWeight.w700,
              letterSpacing: -2,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'Start a conversation',
            style: theme.textTheme.titleMedium?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'Send a message to your agent',
            style: theme.textTheme.bodySmall?.copyWith(
              color: colorScheme.onSurfaceVariant.withValues(alpha: 0.7),
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildErrorState(String message) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.error_outline,
            size: 48,
            color: colorScheme.error.withValues(alpha: 0.7),
          ),
          const SizedBox(height: 16),
          Text(
            'Failed to load messages',
            style: theme.textTheme.titleMedium?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 4),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: Text(
              message,
              style: theme.textTheme.bodySmall?.copyWith(
                color: colorScheme.error,
              ),
              textAlign: TextAlign.center,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMessageWidget(ChatMessage message) {
    return switch (message.type) {
      ChatMessageType.userMessage => UserMessageBubble(message: message),
      ChatMessageType.agentMessage => AgentMessageBubble(message: message),
      ChatMessageType.toolCall => ToolCallWidget(message: message),
      ChatMessageType.toolResult => ToolCallWidget(message: message),
      ChatMessageType.planUpdate => PlanWidget(
        message: message,
        sessionId: widget.sessionId,
      ),
      ChatMessageType.askUser => AskUserWidget(
        message: message,
        sessionId: widget.sessionId,
      ),
      ChatMessageType.reasoning => ReasoningWidget(message: message),
      ChatMessageType.systemMessage =>
        _isLifecycleMessage(message)
            ? AgentLifecycleWidget(message: message)
            : _buildSystemMessage(message),
    };
  }

  bool _isLifecycleMessage(ChatMessage message) {
    return message.id.startsWith('lifecycle-');
  }

  Widget _buildSystemMessage(ChatMessage message) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Text(
        message.content,
        style: theme.textTheme.bodySmall?.copyWith(
          color: colorScheme.onSurfaceVariant,
          fontStyle: FontStyle.italic,
        ),
      ),
    );
  }
}

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_colors.dart';
import '../../application/chat_provider.dart';

/// Bottom input bar with branded styling.
class ChatInputBar extends ConsumerStatefulWidget {
  const ChatInputBar({super.key, required this.sessionId, this.enabled = true});

  final String sessionId;

  /// When false, the input field and send button are disabled.
  /// Used to prevent sending messages when viewing a sub-agent's messages.
  final bool enabled;

  @override
  ConsumerState<ChatInputBar> createState() => _ChatInputBarState();
}

class _ChatInputBarState extends ConsumerState<ChatInputBar> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _send([String? _]) {
    if (!widget.enabled) return;

    final text = _controller.text.trim();
    if (text.isEmpty) return;

    ref.read(chatMessagesProvider(widget.sessionId).notifier).sendMessage(text);
    _controller.clear();
  }

  bool get _canSend => widget.enabled && _controller.text.trim().isNotEmpty;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Container(
      constraints: const BoxConstraints(minHeight: 56),
      color: isDark ? AppColors.dark : AppColors.light,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            Expanded(
              child: TextField(
                controller: _controller,
                enabled: widget.enabled,
                decoration: InputDecoration(
                  hintText: widget.enabled
                      ? 'Ask your agent...'
                      : 'Switch to Supervisor to send messages',
                  hintStyle: TextStyle(color: AppColors.shade3),
                  filled: true,
                  fillColor: isDark ? AppColors.darkAlt : AppColors.white,
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(20),
                    borderSide: BorderSide.none,
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(20),
                    borderSide: BorderSide.none,
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(20),
                    borderSide: BorderSide.none,
                  ),
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 10,
                  ),
                ),
                maxLines: null,
                textInputAction: TextInputAction.send,
                onSubmitted: _send,
                onChanged: (_) => setState(() {}),
              ),
            ),
            const SizedBox(width: 8),
            _buildSendButton(),
          ],
        ),
      ),
    );
  }

  Widget _buildSendButton() {
    return SizedBox(
      width: 40,
      height: 40,
      child: IconButton.filled(
        onPressed: _canSend ? () => _send() : null,
        icon: const Icon(Icons.arrow_upward, size: 20),
        style: IconButton.styleFrom(
          backgroundColor: _canSend
              ? AppColors.accent
              : AppColors.shade3.withValues(alpha: 0.12),
          foregroundColor: _canSend
              ? AppColors.light
              : AppColors.shade3.withValues(alpha: 0.38),
          shape: const CircleBorder(),
          padding: EdgeInsets.zero,
        ),
      ),
    );
  }
}

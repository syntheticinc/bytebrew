import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../application/settings_provider.dart';

/// A group of switch toggles for notification preferences.
class NotificationToggles extends ConsumerWidget {
  const NotificationToggles({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final prefs = ref.watch(notificationPrefsProvider);

    return Column(
      children: [
        SwitchListTile(
          title: const Text('Ask User prompts'),
          subtitle: const Text('Notify when agent needs input'),
          value: prefs.askUser,
          onChanged: (_) {
            ref.read(notificationPrefsProvider.notifier).toggleAskUser();
          },
        ),
        SwitchListTile(
          title: const Text('Task completed'),
          subtitle: const Text('Notify when task finishes'),
          value: prefs.taskCompleted,
          onChanged: (_) {
            ref.read(notificationPrefsProvider.notifier).toggleTaskCompleted();
          },
        ),
        SwitchListTile(
          title: const Text('Errors'),
          subtitle: const Text('Notify on agent errors'),
          value: prefs.errors,
          onChanged: (_) {
            ref.read(notificationPrefsProvider.notifier).toggleErrors();
          },
        ),
      ],
    );
  }
}

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/providers/shared_preferences_provider.dart';
import 'package:bytebrew_mobile/features/settings/domain/settings_repository.dart';
import 'package:bytebrew_mobile/features/settings/infrastructure/local_settings_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'settings_provider.g.dart';

/// Provides the [SettingsRepository] implementation backed by
/// [SharedPreferences].
@Riverpod(keepAlive: true)
SettingsRepository settingsRepository(Ref ref) {
  final prefs = ref.watch(sharedPreferencesProvider);
  return LocalSettingsRepository(prefs);
}

/// Returns all paired servers.
@riverpod
List<Server> servers(Ref ref) {
  final repo = ref.watch(settingsRepositoryProvider);
  return repo.getServers();
}

/// Notification preferences with toggle methods.
@Riverpod(keepAlive: true)
class NotificationPrefs extends _$NotificationPrefs {
  @override
  NotificationSettings build() => const NotificationSettings();

  /// Toggles the ask-user notification preference.
  void toggleAskUser() {
    state = state.copyWith(askUser: !state.askUser);
  }

  /// Toggles the task-completed notification preference.
  void toggleTaskCompleted() {
    state = state.copyWith(taskCompleted: !state.taskCompleted);
  }

  /// Toggles the errors notification preference.
  void toggleErrors() {
    state = state.copyWith(errors: !state.errors);
  }
}

/// User notification preferences.
class NotificationSettings {
  const NotificationSettings({
    this.askUser = true,
    this.taskCompleted = true,
    this.errors = true,
  });

  final bool askUser;
  final bool taskCompleted;
  final bool errors;

  NotificationSettings copyWith({
    bool? askUser,
    bool? taskCompleted,
    bool? errors,
  }) {
    return NotificationSettings(
      askUser: askUser ?? this.askUser,
      taskCompleted: taskCompleted ?? this.taskCompleted,
      errors: errors ?? this.errors,
    );
  }
}

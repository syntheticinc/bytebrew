import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  // =========================================================================
  // NotificationPrefs
  // =========================================================================
  group('NotificationPrefs', () {
    test('initial state has all notifications enabled', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final prefs = container.read(notificationPrefsProvider);

      expect(prefs.askUser, isTrue);
      expect(prefs.taskCompleted, isTrue);
      expect(prefs.errors, isTrue);
    });

    test('toggleAskUser flips askUser flag', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final notifier = container.read(notificationPrefsProvider.notifier);
      notifier.toggleAskUser();

      final prefs = container.read(notificationPrefsProvider);
      expect(prefs.askUser, isFalse);
      expect(prefs.taskCompleted, isTrue);
      expect(prefs.errors, isTrue);
    });

    test('toggleTaskCompleted flips taskCompleted flag', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final notifier = container.read(notificationPrefsProvider.notifier);
      notifier.toggleTaskCompleted();

      final prefs = container.read(notificationPrefsProvider);
      expect(prefs.askUser, isTrue);
      expect(prefs.taskCompleted, isFalse);
      expect(prefs.errors, isTrue);
    });

    test('toggleErrors flips errors flag', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final notifier = container.read(notificationPrefsProvider.notifier);
      notifier.toggleErrors();

      final prefs = container.read(notificationPrefsProvider);
      expect(prefs.askUser, isTrue);
      expect(prefs.taskCompleted, isTrue);
      expect(prefs.errors, isFalse);
    });

    test('double toggle restores original value', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final notifier = container.read(notificationPrefsProvider.notifier);
      notifier.toggleAskUser();
      notifier.toggleAskUser();

      final prefs = container.read(notificationPrefsProvider);
      expect(prefs.askUser, isTrue);
    });

    test('multiple toggles can disable all notifications', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final notifier = container.read(notificationPrefsProvider.notifier);
      notifier.toggleAskUser();
      notifier.toggleTaskCompleted();
      notifier.toggleErrors();

      final prefs = container.read(notificationPrefsProvider);
      expect(prefs.askUser, isFalse);
      expect(prefs.taskCompleted, isFalse);
      expect(prefs.errors, isFalse);
    });
  });

  // =========================================================================
  // NotificationSettings copyWith
  // =========================================================================
  group('NotificationSettings', () {
    test('copyWith preserves unchanged fields', () {
      const settings = NotificationSettings();
      final copy = settings.copyWith(askUser: false);

      expect(copy.askUser, isFalse);
      expect(copy.taskCompleted, isTrue);
      expect(copy.errors, isTrue);
    });

    test('copyWith overrides all fields', () {
      const settings = NotificationSettings();
      final copy = settings.copyWith(
        askUser: false,
        taskCompleted: false,
        errors: false,
      );

      expect(copy.askUser, isFalse);
      expect(copy.taskCompleted, isFalse);
      expect(copy.errors, isFalse);
    });
  });

  // =========================================================================
  // serversProvider
  // =========================================================================
  group('serversProvider', () {
    test('returns servers from overridden value', () {
      final container = ProviderContainer(
        overrides: [
          serversProvider.overrideWithValue([]),
        ],
      );
      addTearDown(container.dispose);

      final servers = container.read(serversProvider);
      expect(servers, isEmpty);
    });
  });
}

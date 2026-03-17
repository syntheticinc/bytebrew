import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:bytebrew_mobile/features/settings/presentation/widgets/notification_toggles.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('NotificationToggles renders all three toggle titles', (
    tester,
  ) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Ask User prompts'), findsOneWidget);
    expect(find.text('Task completed'), findsOneWidget);
    expect(find.text('Errors'), findsOneWidget);
  });

  testWidgets('NotificationToggles renders subtitles', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Notify when agent needs input'), findsOneWidget);
    expect(find.text('Notify when task finishes'), findsOneWidget);
    expect(find.text('Notify on agent errors'), findsOneWidget);
  });

  testWidgets('All switches are initially on', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    final switches = tester.widgetList<SwitchListTile>(
      find.byType(SwitchListTile),
    );

    for (final switchTile in switches) {
      expect(switchTile.value, isTrue);
    }
  });

  testWidgets('Tapping Ask User toggle turns it off', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    // Tap the first SwitchListTile (Ask User prompts).
    await tester.tap(find.text('Ask User prompts'));
    await tester.pumpAndSettle();

    // Verify the switch changed.
    final switches = tester
        .widgetList<SwitchListTile>(find.byType(SwitchListTile))
        .toList();

    // Ask User should be off.
    expect(switches[0].value, isFalse);
    // Others remain on.
    expect(switches[1].value, isTrue);
    expect(switches[2].value, isTrue);
  });

  testWidgets('Tapping Task completed toggle turns it off', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    await tester.tap(find.text('Task completed'));
    await tester.pumpAndSettle();

    final switches = tester
        .widgetList<SwitchListTile>(find.byType(SwitchListTile))
        .toList();

    expect(switches[0].value, isTrue);
    expect(switches[1].value, isFalse);
    expect(switches[2].value, isTrue);
  });

  testWidgets('Tapping Errors toggle turns it off', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    await tester.tap(find.text('Errors'));
    await tester.pumpAndSettle();

    final switches = tester
        .widgetList<SwitchListTile>(find.byType(SwitchListTile))
        .toList();

    expect(switches[0].value, isTrue);
    expect(switches[1].value, isTrue);
    expect(switches[2].value, isFalse);
  });

  testWidgets('Double-tapping restores original state', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    // Toggle off, then on.
    await tester.tap(find.text('Ask User prompts'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Ask User prompts'));
    await tester.pumpAndSettle();

    final switches = tester
        .widgetList<SwitchListTile>(find.byType(SwitchListTile))
        .toList();

    expect(switches[0].value, isTrue);
  });

  testWidgets('NotificationToggles with overridden initial state', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationPrefsProvider.overrideWithValue(
            const NotificationSettings(
              askUser: false,
              taskCompleted: true,
              errors: false,
            ),
          ),
        ],
        child: const MaterialApp(home: Scaffold(body: NotificationToggles())),
      ),
    );

    await tester.pumpAndSettle();

    final switches = tester
        .widgetList<SwitchListTile>(find.byType(SwitchListTile))
        .toList();

    expect(switches[0].value, isFalse); // askUser
    expect(switches[1].value, isTrue); // taskCompleted
    expect(switches[2].value, isFalse); // errors
  });
}

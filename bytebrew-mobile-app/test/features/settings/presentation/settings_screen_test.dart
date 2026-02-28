import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:bytebrew_mobile/features/settings/presentation/settings_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final _testServers = [
  Server(
    id: 'srv-1',
    name: 'MacBook Pro',
    lanAddress: '192.168.1.50',
    connectionMode: ConnectionMode.lan,
    isOnline: true,
    latencyMs: 5,
    pairedAt: DateTime.now().subtract(const Duration(days: 30)),
  ),
  Server(
    id: 'srv-2',
    name: 'Desktop PC',
    lanAddress: '192.168.1.100',
    bridgeUrl: 'bytebrew.io',
    connectionMode: ConnectionMode.bridge,
    isOnline: false,
    latencyMs: 45,
    pairedAt: DateTime.now().subtract(const Duration(days: 7)),
  ),
];

void main() {
  testWidgets('SettingsScreen renders top section headers', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue(_testServers)],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('SERVERS'), findsOneWidget);
    expect(find.text('BRIDGE'), findsOneWidget);
    expect(find.text('NOTIFICATIONS'), findsOneWidget);
  });

  testWidgets('SettingsScreen renders bottom sections after scrolling', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue(_testServers)],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    // Scroll down to reveal APPEARANCE section and footer
    await tester.scrollUntilVisible(
      find.text('APPEARANCE'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.text('APPEARANCE'), findsOneWidget);
  });

  testWidgets('SettingsScreen renders server names', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue(_testServers)],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('MacBook Pro'), findsOneWidget);
    expect(find.text('Desktop PC'), findsOneWidget);
  });

  testWidgets('SettingsScreen renders "Add Server" button', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue(_testServers)],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Add Server'), findsOneWidget);
  });

  testWidgets('SettingsScreen renders AppBar with title', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue([])],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Settings'), findsOneWidget);
  });

  testWidgets('SettingsScreen renders notification toggles', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue([])],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Ask User prompts'), findsOneWidget);
    expect(find.text('Task completed'), findsOneWidget);
    expect(find.text('Errors'), findsOneWidget);
  });

  testWidgets('SettingsScreen renders footer text after scrolling', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [serversProvider.overrideWithValue([])],
        child: const MaterialApp(home: SettingsScreen()),
      ),
    );

    await tester.pumpAndSettle();

    // Scroll down to reveal footer
    await tester.scrollUntilVisible(
      find.text('About | Privacy | v0.1.0'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.text('About | Privacy | v0.1.0'), findsOneWidget);
  });
}

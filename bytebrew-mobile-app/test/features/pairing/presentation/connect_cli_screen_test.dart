import 'package:bytebrew_mobile/features/chat/application/connection_provider.dart';
import 'package:bytebrew_mobile/features/pairing/presentation/connect_cli_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../helpers/fakes.dart';

void main() {
  testWidgets('renders input field and Connect button', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWith(() => FakeWsConnection()),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.byType(TextField), findsOneWidget);
    expect(find.text('Connect'), findsOneWidget);
  });

  testWidgets('shows Disconnected status initially', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWith(() => FakeWsConnection()),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.text('Disconnected'), findsOneWidget);
  });

  testWidgets('shows Connecting status', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWith(
            () => FakeWsConnection(WsConnectionStatus.connecting),
          ),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.text('Connecting...'), findsOneWidget);
    // Connect button is disabled during connecting -- shows spinner instead.
    expect(find.text('Connect'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });

  testWidgets('shows Connection failed status on error', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWith(
            () => FakeWsConnection(WsConnectionStatus.error),
          ),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.text('Connection failed'), findsOneWidget);
    // Connect button is available to retry.
    expect(find.text('Connect'), findsOneWidget);
  });

  testWidgets('shows Connected status and Disconnect button', (tester) async {
    // Use overrideWithValue so the initial state is connected without
    // triggering ref.listen which calls context.go (requires GoRouter).
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWithValue(WsConnectionStatus.connected),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.text('Connected'), findsOneWidget);
    // When connected the button switches to Disconnect.
    expect(find.text('Disconnect'), findsOneWidget);
    expect(find.text('Connect'), findsNothing);
  });

  testWidgets('shows app bar title', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWith(() => FakeWsConnection()),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.text('Connect to CLI'), findsOneWidget);
  });

  testWidgets('shows instruction text', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          wsConnectionProvider.overrideWith(() => FakeWsConnection()),
        ],
        child: const MaterialApp(home: ConnectCliScreen()),
      ),
    );

    expect(find.text('bytebrew --mobile'), findsOneWidget);
    expect(find.text('Start CLI with mobile support:'), findsOneWidget);
  });
}

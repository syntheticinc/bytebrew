import 'package:bytebrew_mobile/features/pairing/presentation/add_server_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget buildTestWidget() {
    return const ProviderScope(child: MaterialApp(home: AddServerScreen()));
  }

  testWidgets('AddServerScreen renders AppBar with title', (tester) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    expect(find.text('Pair New Server'), findsOneWidget);
  });

  testWidgets('AddServerScreen renders instruction section', (tester) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    expect(find.text('Run in your terminal:'), findsOneWidget);
    expect(find.text('bytebrew mobile-pair'), findsOneWidget);
  });

  testWidgets('AddServerScreen renders QR scan mode by default', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    // QR scan mode is default -- segment button should be present.
    expect(find.text('QR Scan'), findsOneWidget);
    expect(find.text('Manual Code'), findsOneWidget);
  });

  testWidgets('AddServerScreen shows manual code form after switching mode', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    // Switch to Manual Code mode.
    await tester.tap(find.text('Manual Code'));
    await tester.pumpAndSettle();

    expect(find.text('Server Address'), findsOneWidget);
    expect(find.text('Enter the 6-digit pairing code'), findsOneWidget);
    expect(find.text('Connect'), findsOneWidget);
  });

  testWidgets('AddServerScreen shows address hint in manual mode', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Manual Code'));
    await tester.pumpAndSettle();

    expect(find.text('e.g. 192.168.1.5'), findsOneWidget);
  });

  testWidgets('Connect button is disabled when form is incomplete', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Manual Code'));
    await tester.pumpAndSettle();

    // Find the Connect button -- it should be disabled (onPressed == null).
    final connectButton = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Connect'),
    );
    expect(connectButton.onPressed, isNull);
  });

  testWidgets('AddServerScreen renders security info section', (tester) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    // Scroll to make security info visible if needed.
    await tester.scrollUntilVisible(
      find.text('Encrypted connection via bridge relay'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.text('Encrypted connection via bridge relay'), findsOneWidget);
    expect(
      find.text('End-to-end encrypted connection through a secure relay'),
      findsOneWidget,
    );
  });

  testWidgets('AddServerScreen shows lock icon in security section', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('Encrypted connection via bridge relay'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.lock_outline), findsOneWidget);
  });

  testWidgets('Entering address enables part of the form', (tester) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Manual Code'));
    await tester.pumpAndSettle();

    // Enter a server address.
    final addressField = find.byType(TextField).first;
    await tester.enterText(addressField, '192.168.1.5');
    await tester.pumpAndSettle();

    // Connect should still be disabled -- code fields are empty.
    final connectButton = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Connect'),
    );
    expect(connectButton.onPressed, isNull);
  });
}

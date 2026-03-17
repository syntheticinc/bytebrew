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

    expect(find.text('In ByteBrew CLI, type:'), findsOneWidget);
    expect(find.text('/mobile'), findsOneWidget);
  });

  testWidgets('AddServerScreen renders QR scanner by default', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    expect(find.text('Point your camera at the QR code shown in CLI after typing /mobile'), findsOneWidget);
  });

  testWidgets('AddServerScreen does not show manual code form', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    expect(find.text('Manual Code'), findsNothing);
    expect(find.text('Enter the 6-digit pairing code'), findsNothing);
    expect(find.text('Connect'), findsNothing);
  });

  testWidgets('AddServerScreen renders security info section', (tester) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('End-to-end encrypted connection'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.text('End-to-end encrypted connection'), findsOneWidget);
    expect(
      find.text('QR code contains a one-time cryptographic token'),
      findsOneWidget,
    );
  });

  testWidgets('AddServerScreen shows lock icon in security section', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestWidget());
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('End-to-end encrypted connection'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.lock_outline), findsOneWidget);
  });
}

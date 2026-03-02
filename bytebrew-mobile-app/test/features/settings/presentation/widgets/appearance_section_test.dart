import 'package:bytebrew_mobile/app.dart';
import 'package:bytebrew_mobile/features/settings/presentation/widgets/appearance_section.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AppearanceSection renders theme label and options',
      (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(body: AppearanceSection()),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Theme'), findsOneWidget);
    expect(find.text('System'), findsOneWidget);
    expect(find.text('Light'), findsOneWidget);
    expect(find.text('Dark'), findsOneWidget);
  });

  testWidgets('AppearanceSection renders font size label', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(body: AppearanceSection()),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Font size'), findsOneWidget);
    expect(find.text('Medium'), findsOneWidget);
  });

  testWidgets('AppearanceSection default selection is Dark', (tester) async {
    // AppThemeMode defaults to ThemeMode.dark in build().
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(body: AppearanceSection()),
        ),
      ),
    );

    await tester.pumpAndSettle();

    // The SegmentedButton should have Dark selected.
    // We verify by checking the SegmentedButton widget's selected set.
    final segmentedButton = tester.widget<SegmentedButton<ThemeMode>>(
      find.byType(SegmentedButton<ThemeMode>),
    );
    expect(segmentedButton.selected, {ThemeMode.dark});
  });

  testWidgets('Tapping Light changes theme mode', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(body: AppearanceSection()),
        ),
      ),
    );

    await tester.pumpAndSettle();

    // Tap "Light" segment.
    await tester.tap(find.text('Light'));
    await tester.pumpAndSettle();

    final segmentedButton = tester.widget<SegmentedButton<ThemeMode>>(
      find.byType(SegmentedButton<ThemeMode>),
    );
    expect(segmentedButton.selected, {ThemeMode.light});
  });

  testWidgets('Tapping System changes theme mode', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(body: AppearanceSection()),
        ),
      ),
    );

    await tester.pumpAndSettle();

    await tester.tap(find.text('System'));
    await tester.pumpAndSettle();

    final segmentedButton = tester.widget<SegmentedButton<ThemeMode>>(
      find.byType(SegmentedButton<ThemeMode>),
    );
    expect(segmentedButton.selected, {ThemeMode.system});
  });

  testWidgets('Theme mode persists across rebuilds', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(body: AppearanceSection()),
        ),
      ),
    );

    await tester.pumpAndSettle();

    // Change to Light.
    await tester.tap(find.text('Light'));
    await tester.pumpAndSettle();

    // Force rebuild by pumping again.
    await tester.pump();

    final segmentedButton = tester.widget<SegmentedButton<ThemeMode>>(
      find.byType(SegmentedButton<ThemeMode>),
    );
    expect(segmentedButton.selected, {ThemeMode.light});
  });
}

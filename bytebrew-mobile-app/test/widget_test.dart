import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/app.dart';

void main() {
  testWidgets('ByteBrewApp renders splash screen', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: ByteBrewApp()));

    // The initial route is /splash, which shows the ByteBrew branding.
    expect(find.text('Byte Brew'), findsOneWidget);
    expect(find.text('Your AI agents, everywhere'), findsOneWidget);

    // Pump past the navigation timer. Cannot use pumpAndSettle because
    // after navigation, AnimatedStatusIndicator has an infinite repeating
    // animation on sessions with needsAttention status.
    await tester.pump(const Duration(seconds: 2));
    await tester.pump(const Duration(milliseconds: 100));
  });
}

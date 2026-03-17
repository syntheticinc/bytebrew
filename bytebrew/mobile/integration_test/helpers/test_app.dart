import 'package:bytebrew_mobile/app.dart';
import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:integration_test/integration_test.dart';

// Import Override for use in this file, and re-export for test files.
import 'package:flutter_riverpod/misc.dart' show Override;
export 'package:flutter_riverpod/misc.dart' show Override;

import '../../test/helpers/fakes.dart';

/// Initializes the integration test binding.
///
/// Must be called at the start of every integration test `main()` before
/// any `testWidgets` calls.
void initializeBinding() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
}

/// Builds the full [ByteBrewApp] wrapped in a [ProviderScope] with
/// sensible defaults for E2E testing.
///
/// [FakeTokenStorage] is always injected to avoid platform channel errors.
/// Pass additional [overrides] to inject fakes for auth, sessions, chat, etc.
Widget buildE2EApp({List<Override> overrides = const []}) {
  return ProviderScope(
    overrides: [
      tokenStorageProvider.overrideWithValue(FakeTokenStorage()),
      ...overrides,
    ],
    child: const ByteBrewApp(),
  );
}

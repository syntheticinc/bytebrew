import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'fakes.dart';

// Re-export Override type so callers can use it without extra imports.
// ignore: depend_on_referenced_packages
export 'package:riverpod/misc.dart' show Override;

/// Builds a fully-configured test application wrapped in a [ProviderScope].
///
/// This helper eliminates boilerplate when testing screens that depend on
/// common providers.  Pass [overrides] to add or replace any provider, or
/// use the convenience parameters [authState] and [sessions] for the most
/// common cases.
///
/// The returned widget includes a [MaterialApp] wrapping [child], suitable
/// for `pumpWidget`.
Widget buildTestApp({
  required Widget child,
  AuthState? authState,
  List<Session>? sessions,
  // ignore: depend_on_referenced_packages
  List<Object> overrides = const [],
}) {
  final allOverrides = [
    if (authState != null) authProvider.overrideWithValue(authState),
    if (sessions != null)
      sessionsProvider.overrideWith(() => FakeSessionsNotifier(sessions)),
    ...overrides,
  ];

  return ProviderScope(
    overrides: allOverrides,
    child: MaterialApp(home: child),
  );
}

import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/auth/presentation/login_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../helpers/fakes.dart';

/// Builds a GoRouter-backed test app so that LoginScreen's
/// `context.go('/sessions')` on successful auth does not crash.
Widget _buildLoginWithRouter({
  AuthState? initialAuthState,
  FakeAuthRepository? authRepo,
}) {
  final router = GoRouter(
    initialLocation: '/login',
    routes: [
      GoRoute(path: '/login', builder: (context, state) => const LoginScreen()),
      GoRoute(
        path: '/sessions',
        builder: (context, state) =>
            const Scaffold(body: Center(child: Text('Sessions placeholder'))),
      ),
    ],
  );

  return ProviderScope(
    overrides: [
      if (initialAuthState != null)
        authProvider.overrideWithValue(initialAuthState),
      tokenStorageProvider.overrideWithValue(FakeTokenStorage()),
      authRepositoryProvider.overrideWithValue(
        authRepo ?? FakeAuthRepository(),
      ),
    ],
    child: MaterialApp.router(routerConfig: router),
  );
}

/// Builds a simple test app (no GoRouter) for tests that only test
/// static state without triggering auth actions.
Widget _buildLoginStatic({AuthState? authState}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWithValue(
        authState ?? const AuthState.unauthenticated(),
      ),
      tokenStorageProvider.overrideWithValue(FakeTokenStorage()),
      authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
    ],
    child: const MaterialApp(home: LoginScreen()),
  );
}

void main() {
  group('Auth flow integration', () {
    testWidgets('TC-AUTH-01: Login success shows no error', (tester) async {
      // Use GoRouter-backed app because LoginScreen calls context.go
      // on auth state change.
      await tester.pumpWidget(
        _buildLoginWithRouter(
          authRepo: FakeAuthRepository(shouldSucceed: true),
        ),
      );

      // Wait for initial state resolution (_checkSavedTokens).
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Enter credentials.
      await tester.enterText(find.byType(TextField).first, 'test@example.com');
      await tester.enterText(find.byType(TextField).last, 'password123');
      await tester.pump();

      // Tap Sign In.
      await tester.tap(find.text('Sign In'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // No error text should be visible.
      expect(find.textContaining('Auth failed'), findsNothing);
      expect(find.textContaining('Exception'), findsNothing);
    });

    testWidgets('TC-AUTH-02: Login failure shows error message', (
      tester,
    ) async {
      await tester.pumpWidget(
        _buildLoginWithRouter(
          authRepo: FakeAuthRepository(
            shouldSucceed: false,
            errorMessage: 'Invalid credentials',
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Enter credentials.
      await tester.enterText(find.byType(TextField).first, 'wrong@test.com');
      await tester.enterText(find.byType(TextField).last, 'wrongpass');
      await tester.pump();

      // Tap Sign In.
      await tester.tap(find.text('Sign In'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Error message should appear (inline text or SnackBar).
      expect(find.textContaining('Invalid credentials'), findsWidgets);
    });

    testWidgets('TC-AUTH-03: Register mode switches form', (tester) async {
      await tester.pumpWidget(_buildLoginStatic());
      await tester.pump();

      // Initially in login mode.
      expect(find.text('Sign In'), findsOneWidget);
      expect(find.text('Create Account'), findsNothing);

      // Tap toggle to register mode.
      await tester.tap(find.text("Don't have an account? Register"));
      await tester.pump();

      // Register mode active.
      expect(find.text('Create Account'), findsOneWidget);
      expect(find.text('Sign In'), findsNothing);
      expect(find.text('Already have an account? Sign in'), findsOneWidget);

      // Toggle back to login mode.
      await tester.tap(find.text('Already have an account? Sign in'));
      await tester.pump();

      expect(find.text('Sign In'), findsOneWidget);
      expect(find.text('Create Account'), findsNothing);
    });

    testWidgets('TC-AUTH-04: Loading state shows progress indicator', (
      tester,
    ) async {
      await tester.pumpWidget(
        _buildLoginStatic(authState: const AuthState.loading()),
      );
      await tester.pump();

      // CircularProgressIndicator should be visible inside the button.
      expect(find.byType(CircularProgressIndicator), findsOneWidget);

      // "Sign In" text should not be visible when loading.
      expect(find.text('Sign In'), findsNothing);
    });
  });
}

import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/auth/presentation/login_screen.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/sessions_screen.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../test/helpers/fakes.dart';
import 'helpers/test_app.dart';

void main() {
  initializeBinding();

  group('TC-E2E-AUTH: Full authentication flow', () {
    testWidgets(
      'TC-E2E-AUTH-01: Splash -> login -> fill form -> Sign In -> sessions',
      (tester) async {
        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authRepositoryProvider.overrideWithValue(
                FakeAuthRepository(shouldSucceed: true),
              ),
              sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
              groupedSessionsProvider.overrideWithValue({}),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Splash screen renders first.
        await tester.pump();
        expect(find.text('Byte Brew'), findsOneWidget);
        expect(find.text('Your AI agents, everywhere'), findsOneWidget);

        // Wait for splash delay (1200ms) + auth check (FakeTokenStorage
        // returns no tokens -> unauthenticated -> redirect to /login).
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        // Should now be on LoginScreen.
        expect(find.byType(LoginScreen), findsOneWidget);
        expect(find.text('Email'), findsOneWidget);
        expect(find.text('Password'), findsOneWidget);

        // Fill in email and password.
        await tester.enterText(find.byType(TextField).first, 'user@test.com');
        await tester.enterText(find.byType(TextField).last, 'secret123');
        await tester.pump();

        // Tap Sign In.
        await tester.tap(find.text('Sign In'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // GoRouter redirect fires: authenticated -> /sessions.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        // Should have navigated to SessionsScreen.
        expect(find.byType(SessionsScreen), findsOneWidget);
        expect(find.text('Activity'), findsWidgets);
      },
    );

    testWidgets(
      'TC-E2E-AUTH-02: Login failure shows error and stays on login',
      (tester) async {
        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authRepositoryProvider.overrideWithValue(
                FakeAuthRepository(
                  shouldSucceed: false,
                  errorMessage: 'Invalid credentials',
                ),
              ),
              sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
              groupedSessionsProvider.overrideWithValue({}),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Wait for splash -> login redirect.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        expect(find.byType(LoginScreen), findsOneWidget);

        // Fill in credentials.
        await tester.enterText(find.byType(TextField).first, 'bad@test.com');
        await tester.enterText(find.byType(TextField).last, 'wrongpass');
        await tester.pump();

        // Tap Sign In.
        await tester.tap(find.text('Sign In'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // Error message should be visible (inline or snackbar).
        expect(find.textContaining('Invalid credentials'), findsWidgets);

        // Should still be on LoginScreen.
        expect(find.byType(LoginScreen), findsOneWidget);
      },
    );

    testWidgets(
      'TC-E2E-AUTH-03: Register mode toggle and register flow',
      (tester) async {
        await tester.pumpWidget(
          buildE2EApp(
            overrides: [
              authRepositoryProvider.overrideWithValue(
                FakeAuthRepository(shouldSucceed: true),
              ),
              sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
              groupedSessionsProvider.overrideWithValue({}),
              serversProvider.overrideWithValue([]),
            ],
          ),
        );

        // Wait for splash -> login.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        // Initially in login mode.
        expect(find.text('Sign In'), findsOneWidget);

        // Switch to register mode.
        await tester.tap(find.text("Don't have an account? Register"));
        await tester.pump();

        expect(find.text('Create Account'), findsOneWidget);
        expect(find.text('Sign In'), findsNothing);

        // Fill in credentials.
        await tester.enterText(find.byType(TextField).first, 'new@test.com');
        await tester.enterText(find.byType(TextField).last, 'newpass123');
        await tester.pump();

        // Tap Create Account.
        await tester.tap(find.text('Create Account'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));

        // Wait for navigation.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        // Should navigate to sessions on successful register.
        expect(find.byType(SessionsScreen), findsOneWidget);
      },
    );
  });
}

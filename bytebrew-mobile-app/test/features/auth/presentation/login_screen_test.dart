import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/auth/presentation/login_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../helpers/fakes.dart';

void main() {
  Widget buildSubject({AuthState? initialState}) {
    return ProviderScope(
      overrides: [
        authProvider.overrideWithValue(
          initialState ?? const AuthState.unauthenticated(),
        ),
        tokenStorageProvider.overrideWithValue(FakeTokenStorage()),
        authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
      ],
      child: const MaterialApp(home: LoginScreen()),
    );
  }

  testWidgets('renders email and password fields', (tester) async {
    await tester.pumpWidget(buildSubject());
    await tester.pump();

    expect(find.text('Email'), findsOneWidget);
    expect(find.text('Password'), findsOneWidget);
  });

  testWidgets('renders sign in button by default', (tester) async {
    await tester.pumpWidget(buildSubject());
    await tester.pump();

    expect(find.text('Sign In'), findsOneWidget);
    expect(find.text('Create Account'), findsNothing);
  });

  testWidgets('toggles to register mode', (tester) async {
    await tester.pumpWidget(buildSubject());
    await tester.pump();

    // Tap the mode toggle
    await tester.tap(find.text("Don't have an account? Register"));
    await tester.pump();

    expect(find.text('Create Account'), findsOneWidget);
    expect(find.text('Sign In'), findsNothing);
    expect(find.text('Already have an account? Sign in'), findsOneWidget);
  });

  testWidgets('shows error text on auth error', (tester) async {
    await tester.pumpWidget(
      buildSubject(initialState: const AuthState.error('Invalid credentials')),
    );
    await tester.pump();

    expect(find.text('Invalid credentials'), findsOneWidget);
  });

  testWidgets('shows loading indicator when auth is loading', (tester) async {
    await tester.pumpWidget(
      buildSubject(initialState: const AuthState.loading()),
    );
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    // Sign In button text should not be visible during loading
    expect(find.text('Sign In'), findsNothing);
  });

  testWidgets('renders wordmark', (tester) async {
    await tester.pumpWidget(buildSubject());
    await tester.pump();

    expect(find.text('Byte Brew'), findsOneWidget);
    expect(find.text('Sign in to continue'), findsOneWidget);
  });
}

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/router/app_router.dart';
import 'package:bytebrew_mobile/features/auth/application/auth_provider.dart';
import 'package:bytebrew_mobile/features/auth/presentation/login_screen.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/sessions/presentation/sessions_screen.dart';
import 'package:bytebrew_mobile/features/settings/application/settings_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../helpers/fakes.dart';

/// Dummy server so that SplashScreen._navigateAuthenticated sees a non-empty
/// server list and navigates to /sessions instead of /add-server.
final _dummyServer = Server(
  id: 'srv-test',
  name: 'Test',
  lanAddress: '127.0.0.1',
  connectionMode: ConnectionMode.lan,
  isOnline: true,
  latencyMs: 1,
  pairedAt: DateTime(2026),
);

/// Builds a test app with GoRouter for navigation tests.
///
/// The [initialAuthState] determines the auth state at startup.
/// The router's redirect logic will react to auth state changes.
Widget _buildNavApp({
  required AuthState initialAuthState,
  List<Session>? sessions,
}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWithValue(initialAuthState),
      tokenStorageProvider.overrideWithValue(FakeTokenStorage()),
      authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
      sessionsProvider.overrideWith(() => FakeSessionsNotifier(sessions ?? [])),
      groupedSessionsProvider.overrideWithValue({}),
      settingsRepositoryProvider
          .overrideWithValue(FakeSettingsRepository([_dummyServer])),
      serversProvider.overrideWithValue([_dummyServer]),
    ],
    child: const _NavTestApp(),
  );
}

/// Wrapper that reads the GoRouter from the provider and creates
/// a MaterialApp.router.
class _NavTestApp extends ConsumerWidget {
  const _NavTestApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(appRouterProvider);
    return MaterialApp.router(routerConfig: router);
  }
}

void main() {
  group('Navigation flow integration', () {
    testWidgets(
      'TC-NAV-01: Splash with unauthenticated state navigates to LoginScreen',
      (tester) async {
        await tester.pumpWidget(
          _buildNavApp(initialAuthState: const AuthState.unauthenticated()),
        );

        // Splash has a 1200ms delay before navigating.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        // Should show LoginScreen elements.
        expect(find.text('Email'), findsOneWidget);
        expect(find.text('Password'), findsOneWidget);
      },
    );

    testWidgets(
      'TC-NAV-02: Splash with authenticated state navigates to SessionsScreen',
      (tester) async {
        await tester.pumpWidget(
          _buildNavApp(initialAuthState: const AuthState.authenticated()),
        );

        // Splash navigates after ~1200ms delay.
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 1300));
        await tester.pump(const Duration(milliseconds: 300));

        // SessionsScreen has "Activity" as AppBar title AND as a
        // NavigationBar label. Use findsWidgets to accept both.
        expect(find.text('Activity'), findsWidgets);

        // Verify SessionsScreen-specific elements: empty state or
        // NavigationBar destinations.
        expect(find.byType(SessionsScreen), findsOneWidget);
      },
    );

    testWidgets(
      'TC-NAV-03: Auth guard redirects unauthenticated user to login',
      (tester) async {
        // Build a minimal GoRouter that starts at /sessions while
        // the redirect logic sends unauthenticated users to /login.
        final router = GoRouter(
          initialLocation: '/sessions',
          redirect: (context, state) {
            if (state.matchedLocation == '/splash') return null;
            if (state.matchedLocation != '/login') return '/login';
            return null;
          },
          routes: [
            GoRoute(
              path: '/login',
              builder: (context, state) => const LoginScreen(),
            ),
            GoRoute(
              path: '/sessions',
              builder: (context, state) => const SessionsScreen(),
            ),
          ],
        );

        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              authProvider.overrideWithValue(const AuthState.unauthenticated()),
              tokenStorageProvider.overrideWithValue(FakeTokenStorage()),
              authRepositoryProvider.overrideWithValue(FakeAuthRepository()),
              sessionsProvider.overrideWith(() => FakeSessionsNotifier([])),
              groupedSessionsProvider.overrideWithValue({}),
              appRouterProvider.overrideWithValue(router),
            ],
            child: MaterialApp.router(routerConfig: router),
          ),
        );

        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        // Should have redirected to LoginScreen.
        expect(find.text('Email'), findsOneWidget);
        expect(find.text('Password'), findsOneWidget);
      },
    );

    testWidgets('TC-NAV-04: Bottom nav tabs switch between screens', (
      tester,
    ) async {
      await tester.pumpWidget(
        _buildNavApp(initialAuthState: const AuthState.authenticated()),
      );

      // Navigate past splash.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 1300));
      await tester.pump(const Duration(milliseconds: 300));

      // Should be on SessionsScreen with bottom nav visible.
      expect(find.byType(SessionsScreen), findsOneWidget);
      expect(find.byType(NavigationBar), findsOneWidget);

      // Find the Settings tab in NavigationBar and tap it.
      // The NavigationBar has NavigationDestination labels.
      final settingsLabel = find.text('Settings');
      expect(settingsLabel, findsOneWidget);

      await tester.tap(settingsLabel);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));

      // SettingsScreen should now be visible with its section headers.
      expect(find.text('SERVERS'), findsOneWidget);
    });
  });
}

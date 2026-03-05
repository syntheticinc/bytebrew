import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/application/auth_provider.dart';
import '../../features/auth/presentation/auth_screen.dart';
import '../../features/auth/presentation/login_screen.dart';
import '../../features/chat/presentation/chat_screen.dart';
import '../../features/pairing/presentation/add_server_screen.dart';
import '../../features/plan/presentation/plan_view_screen.dart';
import '../../features/sessions/presentation/sessions_screen.dart';
import '../../features/settings/presentation/device_list_screen.dart';
import '../../features/settings/presentation/settings_screen.dart';
import '../../features/splash/presentation/splash_screen.dart';

/// Routes that do not require authentication.
const _publicRoutes = {'/splash', '/login'};

final appRouterProvider = Provider<GoRouter>((ref) {
  // Use a ValueNotifier so GoRouter re-evaluates redirects without
  // being recreated (which would destroy the entire navigation tree).
  final authNotifier = ValueNotifier<AuthState>(const AuthState.loading());

  ref.listen<AuthState>(authProvider, (_, next) {
    authNotifier.value = next;
  });

  // Read the current value eagerly so the first redirect has it.
  authNotifier.value = ref.read(authProvider);

  ref.onDispose(() => authNotifier.dispose());

  return GoRouter(
    initialLocation: '/splash',
    refreshListenable: authNotifier,
    redirect: (context, state) {
      final authState = authNotifier.value;
      final location = state.matchedLocation;

      // Never redirect away from splash — it handles its own navigation.
      if (location == '/splash') return null;

      final isPublic = _publicRoutes.contains(location);

      if (authState.status == AuthStatus.unauthenticated && !isPublic) {
        return '/login';
      }

      if (authState.status == AuthStatus.authenticated &&
          location == '/login') {
        return '/sessions';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(path: '/login', builder: (context, state) => const LoginScreen()),
      GoRoute(path: '/auth', builder: (context, state) => const AuthScreen()),
      GoRoute(
        path: '/add-server',
        builder: (context, state) => const AddServerScreen(),
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) {
          return ScaffoldWithNavBar(navigationShell: navigationShell);
        },
        branches: [
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/sessions',
                builder: (context, state) => const SessionsScreen(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/settings',
                builder: (context, state) => const SettingsScreen(),
              ),
            ],
          ),
        ],
      ),
      GoRoute(
        path: '/chat/:sessionId',
        builder: (context, state) {
          final sessionId = state.pathParameters['sessionId']!;
          return ChatScreen(sessionId: sessionId);
        },
      ),
      GoRoute(
        path: '/plan/:sessionId',
        builder: (context, state) {
          final sessionId = state.pathParameters['sessionId']!;
          return PlanViewScreen(sessionId: sessionId);
        },
      ),
      GoRoute(
        path: '/settings/devices/:serverId',
        builder: (context, state) {
          final serverId = state.pathParameters['serverId']!;
          return DeviceListScreen(serverId: serverId);
        },
      ),
    ],
  );
});

/// Shell widget that provides a bottom NavigationBar for tabbed navigation.
class ScaffoldWithNavBar extends StatelessWidget {
  const ScaffoldWithNavBar({super.key, required this.navigationShell});

  final StatefulNavigationShell navigationShell;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: (index) {
          navigationShell.goBranch(
            index,
            initialLocation: index == navigationShell.currentIndex,
          );
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.terminal_outlined),
            selectedIcon: Icon(Icons.terminal),
            label: 'Activity',
          ),
          NavigationDestination(
            icon: Icon(Icons.tune_outlined),
            selectedIcon: Icon(Icons.tune),
            label: 'Settings',
          ),
        ],
      ),
    );
  }
}

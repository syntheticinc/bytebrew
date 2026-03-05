import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'core/infrastructure/ws/ws_connection.dart';
import 'core/router/app_router.dart';
import 'core/theme/app_theme.dart';

part 'app.g.dart';

/// Notifier that manages the current [ThemeMode].
@Riverpod(keepAlive: true)
class AppThemeMode extends _$AppThemeMode {
  @override
  ThemeMode build() => ThemeMode.dark;

  void setThemeMode(ThemeMode mode) {
    state = mode;
  }
}

/// Root application widget.
///
/// Uses [ConsumerStatefulWidget] to listen for app lifecycle changes and
/// gracefully disconnect/reconnect WebSocket connections on
/// background/foreground transitions.
class ByteBrewApp extends ConsumerStatefulWidget {
  const ByteBrewApp({super.key});

  @override
  ConsumerState<ByteBrewApp> createState() => _ByteBrewAppState();
}

class _ByteBrewAppState extends ConsumerState<ByteBrewApp> {
  late final AppLifecycleListener _lifecycleListener;

  @override
  void initState() {
    super.initState();
    _lifecycleListener = AppLifecycleListener(
      onStateChange: _handleLifecycleChange,
    );
  }

  @override
  void dispose() {
    _lifecycleListener.dispose();
    super.dispose();
  }

  void _handleLifecycleChange(AppLifecycleState state) {
    final wsConnection = ref.read(wsConnectionProvider.notifier);
    switch (state) {
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
        wsConnection.disconnect();
      case AppLifecycleState.resumed:
        // Reconnect will happen via auto-connect.
        break;
      case _:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(appRouterProvider);
    final themeMode = ref.watch(appThemeModeProvider);

    GoogleFonts.config.allowRuntimeFetching = false;

    return MaterialApp.router(
      title: 'ByteBrew',
      theme: AppTheme.lightTheme(),
      darkTheme: AppTheme.darkTheme(),
      themeMode: themeMode,
      routerConfig: router,
    );
  }
}

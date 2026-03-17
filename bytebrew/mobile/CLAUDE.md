# ByteBrew Mobile App (Flutter)

## Stack
- Flutter 3.41, Dart 3.x (null-safe, strict)
- State Management: Riverpod 3.x + riverpod_annotation 4.x + riverpod_generator (code gen)
- Navigation: GoRouter 17.x
- UI: Material 3 (useMaterial3: true, ColorScheme.fromSeed)
- API: bytebrew-cloud-api (REST)

## Architecture (Feature-first Clean Architecture)
```
bytebrew-mobile-app/lib/
├── core/                       # Shared: theme, utils, router, widgets, domain
│   ├── domain/                 # Shared entities (Server, Session, ChatMessage, etc.)
│   ├── theme/                  # AppColors, AppTheme
│   ├── utils/                  # timeAgo, etc.
│   ├── widgets/                # StatusIndicator, shared components
│   └── router/                 # GoRouter config (app_router.dart)
├── features/                   # Feature modules
│   ├── splash/presentation/    # SplashScreen
│   ├── pairing/                # domain/ (PairingRepository), infrastructure/ (mock), presentation/
│   ├── sessions/               # domain/ (SessionRepository), infrastructure/ (mock), presentation/
│   ├── chat/                   # domain/ (ChatRepository), infrastructure/ (mock), presentation/
│   ├── plan/presentation/      # PlanViewScreen
│   └── settings/               # domain/ (SettingsRepository), infrastructure/ (mock), presentation/
├── mock/                       # Mock data fixtures (servers, sessions, chat messages)
├── app.dart                    # ByteBrewApp root widget + AppThemeMode notifier
└── main.dart                   # Entry point with ProviderScope
```

Dependencies: `Presentation -> Application -> Domain <- Infrastructure`

## Commands
```bash
flutter run                                              # Run app
dart run build_runner build --delete-conflicting-outputs  # Code gen (riverpod)
flutter test                                             # Unit + widget tests
flutter test integration_test/                           # Integration tests
dart format .                                            # Format
dart analyze                                             # Lint
flutter pub get                                          # Install deps
```

## Key Rules
- `final` by default
- `const` constructors everywhere possible
- Named parameters for >2 arguments
- One widget = one file
- Business logic NOT in widgets -- through application layer
- Early returns (like Go)
- Typed errors (sealed classes)
- `@riverpod` (lowercase) for auto-dispose providers
- `@Riverpod(keepAlive: true)` for persistent providers
- GoRouter provider: plain `Provider<GoRouter>` (no code gen, compatibility issues)
- No `StateProvider` (removed in Riverpod 3) -- use `@riverpod` Notifier instead

## Navigation
- GoRouter with StatefulShellRoute for bottom NavigationBar
- Routes: /splash, /add-server, /sessions, /settings (tabbed), /chat/:sessionId, /plan/:sessionId
- ScaffoldWithNavBar wraps tabbed routes with NavigationBar

## Testing
- Widget tests: `testWidgets()` + `pumpWidget()` + `find.byType()`
- Unit tests: standard `test` package
- Each new screen = widget test
- Wrap with `ProviderScope` in tests

# Flutter Developer Memory

## Project Structure
- Mobile app: `bytebrew-mobile-app/`
- Feature-first architecture with `core/` and `features/` directories
- Riverpod code gen: `dart run build_runner build --delete-conflicting-outputs`

## Key Patterns

### Deprecated APIs (Flutter 3.41+)
- `Color.withOpacity()` is deprecated -- use `Color.withValues(alpha: value)` instead
- Always use `withValues` in new code to pass `dart analyze` cleanly

### Testing with Infinite Animations
- Widgets with `AnimationController.repeat()` (e.g. `AnimatedStatusIndicator`) cause `pumpAndSettle()` to time out
- Solution: use `await tester.pump()` + `await tester.pump(Duration(...))` instead of `pumpAndSettle()`
- This applies to any test rendering `SessionCard` with `needsAttention` status

### Enum Exhaustiveness
- Dart switch expressions must be exhaustive -- when removing enum values, grep ENTIRE codebase for all references
- `SessionStatus` is used in: domain, theme colors, mock data, mock repository, session cards, session groups, sessions screen, providers, tests
- `statusCompleted` color was also used standalone for offline servers and pending plan steps -- renamed to `statusOffline`

## File Locations
- Theme: `lib/core/theme/app_theme.dart`, `lib/core/theme/app_colors.dart`
- Domain entities: `lib/core/domain/session.dart`, `lib/core/domain/plan.dart`
- Session UI: `lib/features/sessions/presentation/`
- Mock data: `lib/mock/mock_sessions.dart`, `lib/mock/mock_servers.dart`
- Router: `lib/core/router/app_router.dart`
- Tests: `test/` directory mirrors `lib/` structure

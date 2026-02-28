# Flutter Developer Agent Memory

## Riverpod 3.x Key Differences
- `StateProvider` is REMOVED in Riverpod 3.x -- use `@riverpod` Notifier instead
- `flutter_riverpod: ^3.2.1`, `riverpod_annotation: ^4.0.2`, `riverpod_generator: ^4.0.3`
- GoRouter provider: use plain `Provider<GoRouter>((ref) => ...)` -- code gen has compatibility issues
- `@riverpod` (lowercase) = auto-dispose; `@Riverpod(keepAlive: true)` = persistent
- Functional providers: MUST use `Ref ref` (typed) -- `strict_top_level_inference` lint requires it
- Family providers (class-based): parameter passed to `build(String param)`, accessed via `this.param` getter (generated)
- Family providers (functional): second parameter after `Ref ref`, e.g. `myProvider(Ref ref, String id)`
- `Ref` is re-exported from `riverpod_annotation` -- no extra import needed

## Project Structure (Stage 1-3)
- Feature-first: `lib/features/{feature}/{domain,infrastructure,application,presentation}/`
- Core shared: `lib/core/{theme,utils,widgets,router,domain}/`
- Root: `lib/app.dart` (ByteBrewApp + AppThemeMode notifier), `lib/main.dart` (entry)
- Application layer: `lib/features/{feature}/application/*_provider.dart` (Riverpod providers)
- Generated files: `*.g.dart` next to annotated files

## Build Commands
- Code gen: `dart run build_runner build --delete-conflicting-outputs`
- Always run build_runner after adding/modifying `@riverpod` annotations
- Run `dart format .` after creating files (formatter adjusts single-line constructors etc.)

## Testing
- Wrap app in `ProviderScope` in tests
- GoRouter initial location `/splash` -- test shows "Splash" text

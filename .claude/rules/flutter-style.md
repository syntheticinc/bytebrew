---
paths:
  - "bytebrew-mobile-app/**/*.dart"
---

# Flutter/Dart Code Style

## Structure (Clean Architecture)
```
bytebrew-mobile-app/
├── lib/
│   ├── domain/            # Entities, value objects
│   ├── application/       # Use cases, state management
│   ├── infrastructure/    # API clients, storage
│   └── presentation/      # Widgets, pages, themes
├── test/                  # Widget + unit tests
└── integration_test/      # Integration tests
```

## Dart Style
- Effective Dart conventions
- `final` по умолчанию, `var` только если нужна мутабельность
- Named parameters для >2 аргументов
- Early returns как в Go

## State Management (Riverpod)
- `flutter_riverpod` + `riverpod_annotation` + `riverpod_generator`
- `@riverpod` аннотация для провайдеров (code generation)
- `ref.watch()` в виджетах, `ref.read()` для one-off actions
- Override провайдеров в тестах через `ProviderScope.overrides`
- НЕ использовать `StateProvider` — предпочитать `@riverpod` с code gen

## Testing
- Widget tests: `testWidgets()` + `pumpWidget()` + `find.byType()`
- Unit tests: стандартный `test` package
- Integration: `integration_test/` с `patrol` или `integration_test`

## Commands
```bash
cd bytebrew-mobile-app
flutter run                    # Run app
flutter test                   # Unit + widget tests
flutter test integration_test/ # Integration tests
dart format .                  # Format
dart analyze                   # Lint
```

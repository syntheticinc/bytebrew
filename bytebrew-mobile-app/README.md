# ByteBrew Mobile App

Мобильный компаньон для ByteBrew AI Agent. Подключается к CLI-сессии через WebSocket или к Cloud API для авторизации и управления сессиями.

## Требования

- Flutter SDK >= 3.41
- Dart SDK >= 3.11
- Android устройство или эмулятор (iOS в процессе)

## Быстрый старт

### 1. Установить зависимости

```bash
cd bytebrew-mobile-app
flutter pub get
```

### 2. Запустить backend (Cloud API)

Приложению нужен Cloud API сервер для авторизации и управления сессиями.

```bash
# Запустить PostgreSQL (Docker)
cd bytebrew-cloud-api
docker-compose up -d

# Запустить Cloud API сервер (порт 9700)
cd bytebrew-cloud-api
go run ./cmd/server
```

### 3. Проброс порта на устройство

При подключении Android-устройства через USB, `localhost` на телефоне не равен `localhost` на ПК. Нужно пробросить порт:

```bash
adb reverse tcp:9700 tcp:9700
```

> Для эмулятора Android проброс не нужен — `10.0.2.2` автоматически указывает на host machine, но приложение использует `localhost`, поэтому `adb reverse` нужен и для эмулятора.

### 4. Запустить приложение

```bash
flutter run
```

Или выбрать конкретное устройство:

```bash
# Список устройств
flutter devices

# Запуск на конкретном устройстве
flutter run -d <device-id>
```

## Конфигурация

### API URL

В **debug** режиме приложение автоматически использует `http://localhost:9700` (см. `lib/main.dart`).

В **release** режиме используется `https://api.bytebrew.io`.

### Полная цепочка сервисов

```
Mobile App (Flutter)
    │
    ├── Auth/Sessions ──→ Cloud API (localhost:9700) ──→ PostgreSQL (localhost:5636)
    │
    └── Live Chat ──→ CLI WebSocket (ws://<ip>:<port>) ──→ bytebrew-srv (gRPC :60401)
```

Для работы Live Chat нужен запущенный CLI с `--mobile` флагом на том же устройстве или в локальной сети.

## Разработка

### Codegen (Riverpod)

После изменения провайдеров с аннотациями `@riverpod`:

```bash
dart run build_runner build --delete-conflicting-outputs
```

### Форматирование и анализ

```bash
dart format .
dart analyze
```

## Тестирование

### Unit + Widget тесты

```bash
flutter test
```

### Integration тесты (widget-level)

```bash
flutter test test/integration/
```

### E2E тесты (на устройстве/flutter-tester)

```bash
# Каждый файл отдельно (flutter-tester не поддерживает несколько файлов за раз)
flutter test integration_test/auth_e2e_test.dart -d flutter-tester
flutter test integration_test/chat_e2e_test.dart -d flutter-tester
flutter test integration_test/multi_message_e2e_test.dart -d flutter-tester
```

## Структура проекта

```
lib/
├── core/                    # Общее: тема, роутер, виджеты, domain entities
│   ├── domain/              # Server, Session, ChatMessage, AuthTokens
│   ├── router/              # GoRouter (app_router.dart)
│   ├── theme/               # AppColors, AppTheme (Material 3)
│   └── widgets/             # StatusIndicator, общие компоненты
├── features/
│   ├── auth/                # Авторизация (login/register)
│   ├── chat/                # Чат с агентом (сообщения, tool calls, plan)
│   ├── pairing/             # Подключение к CLI (WebSocket)
│   ├── sessions/            # Список сессий
│   ├── settings/            # Настройки, управление серверами
│   └── splash/              # Splash screen
├── app.dart                 # ByteBrewApp root widget
└── main.dart                # Entry point
```

## Troubleshooting

### "Сервер не доступен" при авторизации

1. Проверить что Cloud API запущен: `curl http://localhost:9700/health`
2. Проверить проброс порта: `adb reverse --list`
3. Если проброс пуст — выполнить `adb reverse tcp:9700 tcp:9700`

### Gradle build fails

```bash
cd android && ./gradlew clean
flutter clean
flutter pub get
flutter run
```

### "Install canceled by user" (MIUI/Xiaomi)

Настройки → Дополнительные настройки → Для разработчиков → включить **"Установка через USB"**.

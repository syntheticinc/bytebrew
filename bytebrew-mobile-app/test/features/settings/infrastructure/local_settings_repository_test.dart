import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/settings/infrastructure/local_settings_repository.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late LocalSettingsRepository repo;

  setUp(() async {
    // Mock FlutterSecureStorage method channel so delete/write/read work in tests.
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
          const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
          (methodCall) async => null,
        );

    SharedPreferences.setMockInitialValues({});
    final prefs = await SharedPreferences.getInstance();
    repo = LocalSettingsRepository(prefs);
  });

  test('getServers returns empty list on fresh storage', () {
    final servers = repo.getServers();
    expect(servers, isEmpty);
  });

  test('addServer persists and getServers returns it', () async {
    final server = _makeServer(id: 'srv-1', name: 'Test Server');

    await repo.addServer(server);

    final servers = repo.getServers();
    expect(servers, hasLength(1));
    expect(servers.first.id, 'srv-1');
    expect(servers.first.name, 'Test Server');
    expect(servers.first.bridgeUrl, 'ws://bridge:8080');
    expect(servers.first.isOnline, true);
  });

  test('addServer with existing id replaces the server', () async {
    final original = _makeServer(id: 'srv-1', name: 'Original');
    final updated = _makeServer(id: 'srv-1', name: 'Updated');

    await repo.addServer(original);
    await repo.addServer(updated);

    final servers = repo.getServers();
    expect(servers, hasLength(1));
    expect(servers.first.name, 'Updated');
  });

  test('addServer with different ids adds both', () async {
    final first = _makeServer(id: 'srv-1', name: 'First');
    final second = _makeServer(id: 'srv-2', name: 'Second');

    await repo.addServer(first);
    await repo.addServer(second);

    final servers = repo.getServers();
    expect(servers, hasLength(2));
    expect(servers.map((s) => s.id), containsAll(['srv-1', 'srv-2']));
  });

  test('removeServer removes the correct server', () async {
    final first = _makeServer(id: 'srv-1', name: 'First');
    final second = _makeServer(id: 'srv-2', name: 'Second');

    await repo.addServer(first);
    await repo.addServer(second);
    await repo.removeServer('srv-1');

    final servers = repo.getServers();
    expect(servers, hasLength(1));
    expect(servers.first.id, 'srv-2');
  });

  test('removeServer is no-op for unknown id', () async {
    final server = _makeServer(id: 'srv-1', name: 'Only');

    await repo.addServer(server);
    await repo.removeServer('unknown');

    final servers = repo.getServers();
    expect(servers, hasLength(1));
  });

  test('server with bridgeUrl serializes and deserializes correctly', () async {
    final server = Server(
      id: 'srv-bridge',
      name: 'Bridge Server',
      bridgeUrl: 'wss://bridge.bytebrew.ai',
      isOnline: false,
      latencyMs: 42,
      pairedAt: DateTime.utc(2026, 1, 15, 10, 30),
    );

    await repo.addServer(server);

    final restored = repo.getServers().first;
    expect(restored.id, 'srv-bridge');
    expect(restored.bridgeUrl, 'wss://bridge.bytebrew.ai');
    expect(restored.isOnline, false);
    expect(restored.latencyMs, 42);
    expect(restored.pairedAt, DateTime.utc(2026, 1, 15, 10, 30));
  });

  test('pairedAt preserves time precision', () async {
    final pairedAt = DateTime.utc(2026, 2, 28, 14, 30, 45, 123);
    final server = Server(
      id: 'srv-time',
      name: 'Time Test',
      bridgeUrl: 'ws://bridge:8080',
      isOnline: true,
      latencyMs: 10,
      pairedAt: pairedAt,
    );

    await repo.addServer(server);

    final restored = repo.getServers().first;
    expect(restored.pairedAt, pairedAt);
  });
}

Server _makeServer({required String id, required String name}) => Server(
  id: id,
  name: name,
  bridgeUrl: 'ws://bridge:8080',
  isOnline: true,
  latencyMs: 10,
  pairedAt: DateTime.utc(2026, 1, 1),
);

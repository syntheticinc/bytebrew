import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/pairing/application/pairing_provider.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// ---------------------------------------------------------------------------
// Fake pairing repository
// ---------------------------------------------------------------------------

class _FakePairingRepository implements PairingRepository {
  _FakePairingRepository({this.shouldSucceed = true});

  final bool shouldSucceed;
  String? lastAddress;
  String? lastCode;

  @override
  Future<Server> pair({
    required String serverAddress,
    required String pairingCode,
  }) async {
    lastAddress = serverAddress;
    lastCode = pairingCode;

    if (!shouldSucceed) {
      throw Exception('Pairing failed: invalid code');
    }

    return Server(
      id: 'paired-srv',
      name: 'Paired Server',
      lanAddress: serverAddress,
      connectionMode: ConnectionMode.lan,
      isOnline: true,
      latencyMs: 5,
      pairedAt: DateTime.now(),
      deviceToken: 'token-123',
    );
  }
}

void main() {
  // =========================================================================
  // PairDevice
  // =========================================================================
  group('PairDevice', () {
    test('initial state is AsyncData(null)', () {
      final fakeRepo = _FakePairingRepository();

      final container = ProviderContainer(
        overrides: [
          pairingRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      final state = container.read(pairDeviceProvider);
      expect(state, isA<AsyncData<Server?>>());
      expect(state.value, isNull);
    });

    test('pair() returns server on success', () async {
      final fakeRepo = _FakePairingRepository();

      final container = ProviderContainer(
        overrides: [
          pairingRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      final server = await container.read(pairDeviceProvider.notifier).pair(
        serverAddress: '192.168.1.10',
        pairingCode: '123456',
      );

      expect(server.name, 'Paired Server');
      expect(server.lanAddress, '192.168.1.10');
      expect(server.deviceToken, 'token-123');

      // Verify repository received correct args.
      expect(fakeRepo.lastAddress, '192.168.1.10');
      expect(fakeRepo.lastCode, '123456');

      // State should be AsyncData with the server.
      final state = container.read(pairDeviceProvider);
      expect(state.value, isNotNull);
      expect(state.value!.id, 'paired-srv');
    });

    test('pair() transitions through loading state', () async {
      final fakeRepo = _FakePairingRepository();
      final states = <AsyncValue<Server?>>[];

      final container = ProviderContainer(
        overrides: [
          pairingRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      // Listen to state changes.
      container.listen(pairDeviceProvider, (_, next) {
        states.add(next);
      });

      await container.read(pairDeviceProvider.notifier).pair(
        serverAddress: '192.168.1.10',
        pairingCode: '123456',
      );

      // Should have gone through: loading -> data(server)
      expect(states.any((s) => s is AsyncLoading<Server?>), isTrue);
      expect(
        states.last,
        isA<AsyncData<Server?>>()
            .having((s) => s.value?.id, 'server id', 'paired-srv'),
      );
    });

    test('pair() sets error state on failure', () async {
      final fakeRepo = _FakePairingRepository(shouldSucceed: false);

      final container = ProviderContainer(
        overrides: [
          pairingRepositoryProvider.overrideWithValue(fakeRepo),
        ],
      );
      addTearDown(container.dispose);

      try {
        await container.read(pairDeviceProvider.notifier).pair(
          serverAddress: '192.168.1.10',
          pairingCode: '000000',
        );
        fail('Expected an exception');
      } on Exception catch (e) {
        expect(e.toString(), contains('Pairing failed'));
      }
    });
  });
}

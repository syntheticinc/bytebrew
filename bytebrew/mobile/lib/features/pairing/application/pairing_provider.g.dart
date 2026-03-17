// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'pairing_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Provides the [PairingRepository] implementation backed by WebSocket.

@ProviderFor(pairingRepository)
final pairingRepositoryProvider = PairingRepositoryProvider._();

/// Provides the [PairingRepository] implementation backed by WebSocket.

final class PairingRepositoryProvider
    extends
        $FunctionalProvider<
          PairingRepository,
          PairingRepository,
          PairingRepository
        >
    with $Provider<PairingRepository> {
  /// Provides the [PairingRepository] implementation backed by WebSocket.
  PairingRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'pairingRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$pairingRepositoryHash();

  @$internal
  @override
  $ProviderElement<PairingRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  PairingRepository create(Ref ref) {
    return pairingRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PairingRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PairingRepository>(value),
    );
  }
}

String _$pairingRepositoryHash() => r'9950583c4fe68962a7ef7c73254a7f48e6ffff8d';

/// Manages server pairing state.

@ProviderFor(PairDevice)
final pairDeviceProvider = PairDeviceProvider._();

/// Manages server pairing state.
final class PairDeviceProvider
    extends $AsyncNotifierProvider<PairDevice, Server?> {
  /// Manages server pairing state.
  PairDeviceProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'pairDeviceProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$pairDeviceHash();

  @$internal
  @override
  PairDevice create() => PairDevice();
}

String _$pairDeviceHash() => r'e062ae69037aafc846fb20c816ff4e3c0b9b4242';

/// Manages server pairing state.

abstract class _$PairDevice extends $AsyncNotifier<Server?> {
  FutureOr<Server?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Server?>, Server?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Server?>, Server?>,
              AsyncValue<Server?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

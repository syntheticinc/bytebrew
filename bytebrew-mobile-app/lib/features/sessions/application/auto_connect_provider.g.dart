// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auto_connect_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Automatically connects to all saved servers when first read.
///
/// This provider is kept alive so the connections persist for the app's
/// lifetime. It should be watched from the sessions screen to trigger
/// on first navigation.
///
/// On first run: calls connectToAll which initiates WS connections.
/// After that: periodic check every 30s to reconnect any failed servers.

@ProviderFor(sessionsAutoConnect)
final sessionsAutoConnectProvider = SessionsAutoConnectProvider._();

/// Automatically connects to all saved servers when first read.
///
/// This provider is kept alive so the connections persist for the app's
/// lifetime. It should be watched from the sessions screen to trigger
/// on first navigation.
///
/// On first run: calls connectToAll which initiates WS connections.
/// After that: periodic check every 30s to reconnect any failed servers.

final class SessionsAutoConnectProvider
    extends $FunctionalProvider<AsyncValue<void>, void, FutureOr<void>>
    with $FutureModifier<void>, $FutureProvider<void> {
  /// Automatically connects to all saved servers when first read.
  ///
  /// This provider is kept alive so the connections persist for the app's
  /// lifetime. It should be watched from the sessions screen to trigger
  /// on first navigation.
  ///
  /// On first run: calls connectToAll which initiates WS connections.
  /// After that: periodic check every 30s to reconnect any failed servers.
  SessionsAutoConnectProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'sessionsAutoConnectProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sessionsAutoConnectHash();

  @$internal
  @override
  $FutureProviderElement<void> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<void> create(Ref ref) {
    return sessionsAutoConnect(ref);
  }
}

String _$sessionsAutoConnectHash() =>
    r'6fdc388c70f08f6e85627a7331e0781039523974';

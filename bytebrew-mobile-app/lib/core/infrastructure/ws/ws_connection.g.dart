// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'ws_connection.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// WebSocket client that connects to the CLI MobileProxyServer.
///
/// Protocol:
/// - Incoming: `{"type":"init",...}`, `{"type":"event","event":{...}}`
/// - Outgoing: `{"type":"user_message",...}`, `{"type":"ask_user_answer",...}`,
///             `{"type":"cancel"}`

@ProviderFor(WsConnection)
final wsConnectionProvider = WsConnectionProvider._();

/// WebSocket client that connects to the CLI MobileProxyServer.
///
/// Protocol:
/// - Incoming: `{"type":"init",...}`, `{"type":"event","event":{...}}`
/// - Outgoing: `{"type":"user_message",...}`, `{"type":"ask_user_answer",...}`,
///             `{"type":"cancel"}`
final class WsConnectionProvider
    extends $NotifierProvider<WsConnection, WsConnectionStatus> {
  /// WebSocket client that connects to the CLI MobileProxyServer.
  ///
  /// Protocol:
  /// - Incoming: `{"type":"init",...}`, `{"type":"event","event":{...}}`
  /// - Outgoing: `{"type":"user_message",...}`, `{"type":"ask_user_answer",...}`,
  ///             `{"type":"cancel"}`
  WsConnectionProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'wsConnectionProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$wsConnectionHash();

  @$internal
  @override
  WsConnection create() => WsConnection();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(WsConnectionStatus value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<WsConnectionStatus>(value),
    );
  }
}

String _$wsConnectionHash() => r'1c4b33756603c3aeceb50a222a3ad8d8e0f062e7';

/// WebSocket client that connects to the CLI MobileProxyServer.
///
/// Protocol:
/// - Incoming: `{"type":"init",...}`, `{"type":"event","event":{...}}`
/// - Outgoing: `{"type":"user_message",...}`, `{"type":"ask_user_answer",...}`,
///             `{"type":"cancel"}`

abstract class _$WsConnection extends $Notifier<WsConnectionStatus> {
  WsConnectionStatus build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<WsConnectionStatus, WsConnectionStatus>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<WsConnectionStatus, WsConnectionStatus>,
              WsConnectionStatus,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

/// Whether there is an active WebSocket connection.

@ProviderFor(hasActiveConnection)
final hasActiveConnectionProvider = HasActiveConnectionProvider._();

/// Whether there is an active WebSocket connection.

final class HasActiveConnectionProvider
    extends $FunctionalProvider<bool, bool, bool>
    with $Provider<bool> {
  /// Whether there is an active WebSocket connection.
  HasActiveConnectionProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'hasActiveConnectionProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$hasActiveConnectionHash();

  @$internal
  @override
  $ProviderElement<bool> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  bool create(Ref ref) {
    return hasActiveConnection(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }
}

String _$hasActiveConnectionHash() =>
    r'287c751ce3572c58aa16771a5f8523545fc67054';

// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'settings_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Provides the [SettingsRepository] implementation backed by
/// [SharedPreferences].

@ProviderFor(settingsRepository)
final settingsRepositoryProvider = SettingsRepositoryProvider._();

/// Provides the [SettingsRepository] implementation backed by
/// [SharedPreferences].

final class SettingsRepositoryProvider
    extends
        $FunctionalProvider<
          SettingsRepository,
          SettingsRepository,
          SettingsRepository
        >
    with $Provider<SettingsRepository> {
  /// Provides the [SettingsRepository] implementation backed by
  /// [SharedPreferences].
  SettingsRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'settingsRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$settingsRepositoryHash();

  @$internal
  @override
  $ProviderElement<SettingsRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SettingsRepository create(Ref ref) {
    return settingsRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SettingsRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SettingsRepository>(value),
    );
  }
}

String _$settingsRepositoryHash() =>
    r'65af8bec0116ed16667a441468739f01d8a1fbf4';

/// Returns all paired servers.

@ProviderFor(servers)
final serversProvider = ServersProvider._();

/// Returns all paired servers.

final class ServersProvider
    extends $FunctionalProvider<List<Server>, List<Server>, List<Server>>
    with $Provider<List<Server>> {
  /// Returns all paired servers.
  ServersProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'serversProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$serversHash();

  @$internal
  @override
  $ProviderElement<List<Server>> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  List<Server> create(Ref ref) {
    return servers(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<Server> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<Server>>(value),
    );
  }
}

String _$serversHash() => r'8c51d1f8e25befa7cfcca9885bda62463c9dd343';

/// Notification preferences with toggle methods.

@ProviderFor(NotificationPrefs)
final notificationPrefsProvider = NotificationPrefsProvider._();

/// Notification preferences with toggle methods.
final class NotificationPrefsProvider
    extends $NotifierProvider<NotificationPrefs, NotificationSettings> {
  /// Notification preferences with toggle methods.
  NotificationPrefsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationPrefsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationPrefsHash();

  @$internal
  @override
  NotificationPrefs create() => NotificationPrefs();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationSettings value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationSettings>(value),
    );
  }
}

String _$notificationPrefsHash() => r'4c22226546f1d0d30bbfeb139bc513c52033e23a';

/// Notification preferences with toggle methods.

abstract class _$NotificationPrefs extends $Notifier<NotificationSettings> {
  NotificationSettings build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<NotificationSettings, NotificationSettings>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<NotificationSettings, NotificationSettings>,
              NotificationSettings,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

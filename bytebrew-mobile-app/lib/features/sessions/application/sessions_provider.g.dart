// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'sessions_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Provides the [SessionRepository] backed by gRPC via [ConnectionManager].

@ProviderFor(sessionRepository)
final sessionRepositoryProvider = SessionRepositoryProvider._();

/// Provides the [SessionRepository] backed by gRPC via [ConnectionManager].

final class SessionRepositoryProvider
    extends
        $FunctionalProvider<
          SessionRepository,
          SessionRepository,
          SessionRepository
        >
    with $Provider<SessionRepository> {
  /// Provides the [SessionRepository] backed by gRPC via [ConnectionManager].
  SessionRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'sessionRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sessionRepositoryHash();

  @$internal
  @override
  $ProviderElement<SessionRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SessionRepository create(Ref ref) {
    return sessionRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SessionRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SessionRepository>(value),
    );
  }
}

String _$sessionRepositoryHash() => r'8fe388777fbbadf90b2b2367dcd56a41b5aca898';

/// Manages the list of agent sessions.
///
/// Subscribes to push-based session updates from the server via
/// [SessionRepository.watchSessions]. No polling is needed.

@ProviderFor(Sessions)
final sessionsProvider = SessionsProvider._();

/// Manages the list of agent sessions.
///
/// Subscribes to push-based session updates from the server via
/// [SessionRepository.watchSessions]. No polling is needed.
final class SessionsProvider
    extends $AsyncNotifierProvider<Sessions, List<Session>> {
  /// Manages the list of agent sessions.
  ///
  /// Subscribes to push-based session updates from the server via
  /// [SessionRepository.watchSessions]. No polling is needed.
  SessionsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'sessionsProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sessionsHash();

  @$internal
  @override
  Sessions create() => Sessions();
}

String _$sessionsHash() => r'68eceade26752bd5ffc597610b787402e4599380';

/// Manages the list of agent sessions.
///
/// Subscribes to push-based session updates from the server via
/// [SessionRepository.watchSessions]. No polling is needed.

abstract class _$Sessions extends $AsyncNotifier<List<Session>> {
  FutureOr<List<Session>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<Session>>, List<Session>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<Session>>, List<Session>>,
              AsyncValue<List<Session>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

/// Groups sessions by their [SessionStatus].

@ProviderFor(groupedSessions)
final groupedSessionsProvider = GroupedSessionsProvider._();

/// Groups sessions by their [SessionStatus].

final class GroupedSessionsProvider
    extends
        $FunctionalProvider<
          Map<SessionStatus, List<Session>>,
          Map<SessionStatus, List<Session>>,
          Map<SessionStatus, List<Session>>
        >
    with $Provider<Map<SessionStatus, List<Session>>> {
  /// Groups sessions by their [SessionStatus].
  GroupedSessionsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'groupedSessionsProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$groupedSessionsHash();

  @$internal
  @override
  $ProviderElement<Map<SessionStatus, List<Session>>> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  Map<SessionStatus, List<Session>> create(Ref ref) {
    return groupedSessions(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Map<SessionStatus, List<Session>> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Map<SessionStatus, List<Session>>>(
        value,
      ),
    );
  }
}

String _$groupedSessionsHash() => r'af91c35eef3b28849894ee9c91c51b6a111e7d86';

/// Finds a single session by [id], or null if not found.

@ProviderFor(sessionById)
final sessionByIdProvider = SessionByIdFamily._();

/// Finds a single session by [id], or null if not found.

final class SessionByIdProvider
    extends $FunctionalProvider<Session?, Session?, Session?>
    with $Provider<Session?> {
  /// Finds a single session by [id], or null if not found.
  SessionByIdProvider._({
    required SessionByIdFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'sessionByIdProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$sessionByIdHash();

  @override
  String toString() {
    return r'sessionByIdProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<Session?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Session? create(Ref ref) {
    final argument = this.argument as String;
    return sessionById(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Session? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Session?>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is SessionByIdProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$sessionByIdHash() => r'ab213197583c2d1f21872e2833105eb8002ec9ea';

/// Finds a single session by [id], or null if not found.

final class SessionByIdFamily extends $Family
    with $FunctionalFamilyOverride<Session?, String> {
  SessionByIdFamily._()
    : super(
        retry: null,
        name: r'sessionByIdProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Finds a single session by [id], or null if not found.

  SessionByIdProvider call(String id) =>
      SessionByIdProvider._(argument: id, from: this);

  @override
  String toString() => r'sessionByIdProvider';
}

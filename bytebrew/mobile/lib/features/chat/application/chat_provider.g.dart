// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'chat_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Singleton [ChatMessageStore] for persisting chat messages to SQLite.
///
/// Kept alive to avoid re-opening the database on every chat screen visit.

@ProviderFor(chatMessageStore)
final chatMessageStoreProvider = ChatMessageStoreProvider._();

/// Singleton [ChatMessageStore] for persisting chat messages to SQLite.
///
/// Kept alive to avoid re-opening the database on every chat screen visit.

final class ChatMessageStoreProvider
    extends
        $FunctionalProvider<
          ChatMessageStore,
          ChatMessageStore,
          ChatMessageStore
        >
    with $Provider<ChatMessageStore> {
  /// Singleton [ChatMessageStore] for persisting chat messages to SQLite.
  ///
  /// Kept alive to avoid re-opening the database on every chat screen visit.
  ChatMessageStoreProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'chatMessageStoreProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$chatMessageStoreHash();

  @$internal
  @override
  $ProviderElement<ChatMessageStore> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ChatMessageStore create(Ref ref) {
    return chatMessageStore(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ChatMessageStore value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ChatMessageStore>(value),
    );
  }
}

String _$chatMessageStoreHash() => r'c938e1af9851407d795b0edbb8e5befeb7f815a8';

/// Provides a default [ChatRepository].
///
/// Returns [EmptyChatRepository] by default. Session-specific repositories
/// are created via [sessionChatRepositoryProvider].

@ProviderFor(chatRepository)
final chatRepositoryProvider = ChatRepositoryProvider._();

/// Provides a default [ChatRepository].
///
/// Returns [EmptyChatRepository] by default. Session-specific repositories
/// are created via [sessionChatRepositoryProvider].

final class ChatRepositoryProvider
    extends $FunctionalProvider<ChatRepository, ChatRepository, ChatRepository>
    with $Provider<ChatRepository> {
  /// Provides a default [ChatRepository].
  ///
  /// Returns [EmptyChatRepository] by default. Session-specific repositories
  /// are created via [sessionChatRepositoryProvider].
  ChatRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'chatRepositoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$chatRepositoryHash();

  @$internal
  @override
  $ProviderElement<ChatRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ChatRepository create(Ref ref) {
    return chatRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ChatRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ChatRepository>(value),
    );
  }
}

String _$chatRepositoryHash() => r'851ca9b0dbc912a08188ad92905b26920dd6a909';

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Uses [ref.read] instead of [ref.watch] on [sessionsProvider] to avoid
/// rebuilding (and dropping messages) when the session list refreshes.
/// The serverId is captured once at creation time.
/// Falls back to [EmptyChatRepository] if the session is not found.

@ProviderFor(sessionChatRepository)
final sessionChatRepositoryProvider = SessionChatRepositoryFamily._();

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Uses [ref.read] instead of [ref.watch] on [sessionsProvider] to avoid
/// rebuilding (and dropping messages) when the session list refreshes.
/// The serverId is captured once at creation time.
/// Falls back to [EmptyChatRepository] if the session is not found.

final class SessionChatRepositoryProvider
    extends $FunctionalProvider<ChatRepository, ChatRepository, ChatRepository>
    with $Provider<ChatRepository> {
  /// Resolves the [ChatRepository] for a specific [sessionId].
  ///
  /// Uses [ref.read] instead of [ref.watch] on [sessionsProvider] to avoid
  /// rebuilding (and dropping messages) when the session list refreshes.
  /// The serverId is captured once at creation time.
  /// Falls back to [EmptyChatRepository] if the session is not found.
  SessionChatRepositoryProvider._({
    required SessionChatRepositoryFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'sessionChatRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$sessionChatRepositoryHash();

  @override
  String toString() {
    return r'sessionChatRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<ChatRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ChatRepository create(Ref ref) {
    final argument = this.argument as String;
    return sessionChatRepository(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ChatRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ChatRepository>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is SessionChatRepositoryProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$sessionChatRepositoryHash() =>
    r'84ad6ec6a7abfa96474ef8c103a32c36272a5dca';

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Uses [ref.read] instead of [ref.watch] on [sessionsProvider] to avoid
/// rebuilding (and dropping messages) when the session list refreshes.
/// The serverId is captured once at creation time.
/// Falls back to [EmptyChatRepository] if the session is not found.

final class SessionChatRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<ChatRepository, String> {
  SessionChatRepositoryFamily._()
    : super(
        retry: null,
        name: r'sessionChatRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Resolves the [ChatRepository] for a specific [sessionId].
  ///
  /// Uses [ref.read] instead of [ref.watch] on [sessionsProvider] to avoid
  /// rebuilding (and dropping messages) when the session list refreshes.
  /// The serverId is captured once at creation time.
  /// Falls back to [EmptyChatRepository] if the session is not found.

  SessionChatRepositoryProvider call(String sessionId) =>
      SessionChatRepositoryProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'sessionChatRepositoryProvider';
}

/// Manages chat messages for a given session.
///
/// Automatically selects the right repository implementation based on the
/// WebSocket connection. Subscribes to real-time message streams when
/// available.

@ProviderFor(ChatMessages)
final chatMessagesProvider = ChatMessagesFamily._();

/// Manages chat messages for a given session.
///
/// Automatically selects the right repository implementation based on the
/// WebSocket connection. Subscribes to real-time message streams when
/// available.
final class ChatMessagesProvider
    extends $AsyncNotifierProvider<ChatMessages, List<ChatMessage>> {
  /// Manages chat messages for a given session.
  ///
  /// Automatically selects the right repository implementation based on the
  /// WebSocket connection. Subscribes to real-time message streams when
  /// available.
  ChatMessagesProvider._({
    required ChatMessagesFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'chatMessagesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatMessagesHash();

  @override
  String toString() {
    return r'chatMessagesProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ChatMessages create() => ChatMessages();

  @override
  bool operator ==(Object other) {
    return other is ChatMessagesProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatMessagesHash() => r'88ef88800a5e3262d14946cde8b6c3118d9fa5a4';

/// Manages chat messages for a given session.
///
/// Automatically selects the right repository implementation based on the
/// WebSocket connection. Subscribes to real-time message streams when
/// available.

final class ChatMessagesFamily extends $Family
    with
        $ClassFamilyOverride<
          ChatMessages,
          AsyncValue<List<ChatMessage>>,
          List<ChatMessage>,
          FutureOr<List<ChatMessage>>,
          String
        > {
  ChatMessagesFamily._()
    : super(
        retry: null,
        name: r'chatMessagesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Manages chat messages for a given session.
  ///
  /// Automatically selects the right repository implementation based on the
  /// WebSocket connection. Subscribes to real-time message streams when
  /// available.

  ChatMessagesProvider call(String sessionId) =>
      ChatMessagesProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'chatMessagesProvider';
}

/// Manages chat messages for a given session.
///
/// Automatically selects the right repository implementation based on the
/// WebSocket connection. Subscribes to real-time message streams when
/// available.

abstract class _$ChatMessages extends $AsyncNotifier<List<ChatMessage>> {
  late final _$args = ref.$arg as String;
  String get sessionId => _$args;

  FutureOr<List<ChatMessage>> build(String sessionId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<List<ChatMessage>>, List<ChatMessage>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<ChatMessage>>, List<ChatMessage>>,
              AsyncValue<List<ChatMessage>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

/// Whether the session is currently being processed by the server.
///
/// Emits `true` when a ProcessingStarted event is received and `false`
/// when processing stops (idle / completed / failed).

@ProviderFor(isProcessing)
final isProcessingProvider = IsProcessingFamily._();

/// Whether the session is currently being processed by the server.
///
/// Emits `true` when a ProcessingStarted event is received and `false`
/// when processing stops (idle / completed / failed).

final class IsProcessingProvider
    extends $FunctionalProvider<AsyncValue<bool>, bool, Stream<bool>>
    with $FutureModifier<bool>, $StreamProvider<bool> {
  /// Whether the session is currently being processed by the server.
  ///
  /// Emits `true` when a ProcessingStarted event is received and `false`
  /// when processing stops (idle / completed / failed).
  IsProcessingProvider._({
    required IsProcessingFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'isProcessingProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$isProcessingHash();

  @override
  String toString() {
    return r'isProcessingProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $StreamProviderElement<bool> $createElement($ProviderPointer pointer) =>
      $StreamProviderElement(pointer);

  @override
  Stream<bool> create(Ref ref) {
    final argument = this.argument as String;
    return isProcessing(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is IsProcessingProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$isProcessingHash() => r'a793b009273f9b926b0eee1c938d49aac1e7f0c7';

/// Whether the session is currently being processed by the server.
///
/// Emits `true` when a ProcessingStarted event is received and `false`
/// when processing stops (idle / completed / failed).

final class IsProcessingFamily extends $Family
    with $FunctionalFamilyOverride<Stream<bool>, String> {
  IsProcessingFamily._()
    : super(
        retry: null,
        name: r'isProcessingProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Whether the session is currently being processed by the server.
  ///
  /// Emits `true` when a ProcessingStarted event is received and `false`
  /// when processing stops (idle / completed / failed).

  IsProcessingProvider call(String sessionId) =>
      IsProcessingProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'isProcessingProvider';
}

/// Returns the active plan from the latest planUpdate message, or null.

@ProviderFor(activePlan)
final activePlanProvider = ActivePlanFamily._();

/// Returns the active plan from the latest planUpdate message, or null.

final class ActivePlanProvider
    extends $FunctionalProvider<PlanData?, PlanData?, PlanData?>
    with $Provider<PlanData?> {
  /// Returns the active plan from the latest planUpdate message, or null.
  ActivePlanProvider._({
    required ActivePlanFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'activePlanProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$activePlanHash();

  @override
  String toString() {
    return r'activePlanProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<PlanData?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  PlanData? create(Ref ref) {
    final argument = this.argument as String;
    return activePlan(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PlanData? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PlanData?>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is ActivePlanProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$activePlanHash() => r'372b5976db4fb1d10a4c7515f7f8ebc96fc08e91';

/// Returns the active plan from the latest planUpdate message, or null.

final class ActivePlanFamily extends $Family
    with $FunctionalFamilyOverride<PlanData?, String> {
  ActivePlanFamily._()
    : super(
        retry: null,
        name: r'activePlanProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Returns the active plan from the latest planUpdate message, or null.

  ActivePlanProvider call(String sessionId) =>
      ActivePlanProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'activePlanProvider';
}

/// Returns the pending ask-user message, or null if none.

@ProviderFor(pendingAskUser)
final pendingAskUserProvider = PendingAskUserFamily._();

/// Returns the pending ask-user message, or null if none.

final class PendingAskUserProvider
    extends $FunctionalProvider<ChatMessage?, ChatMessage?, ChatMessage?>
    with $Provider<ChatMessage?> {
  /// Returns the pending ask-user message, or null if none.
  PendingAskUserProvider._({
    required PendingAskUserFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'pendingAskUserProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$pendingAskUserHash();

  @override
  String toString() {
    return r'pendingAskUserProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<ChatMessage?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ChatMessage? create(Ref ref) {
    final argument = this.argument as String;
    return pendingAskUser(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ChatMessage? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ChatMessage?>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is PendingAskUserProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$pendingAskUserHash() => r'f1d82b45117313d7a38bd0d00d41f81acef5e40e';

/// Returns the pending ask-user message, or null if none.

final class PendingAskUserFamily extends $Family
    with $FunctionalFamilyOverride<ChatMessage?, String> {
  PendingAskUserFamily._()
    : super(
        retry: null,
        name: r'pendingAskUserProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Returns the pending ask-user message, or null if none.

  PendingAskUserProvider call(String sessionId) =>
      PendingAskUserProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'pendingAskUserProvider';
}

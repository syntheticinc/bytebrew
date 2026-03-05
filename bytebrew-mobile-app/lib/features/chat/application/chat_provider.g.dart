// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'chat_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Returns the WS-backed repository when connected, [EmptyChatRepository]
/// otherwise.

@ProviderFor(sessionChatRepository)
final sessionChatRepositoryProvider = SessionChatRepositoryFamily._();

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Returns the WS-backed repository when connected, [EmptyChatRepository]
/// otherwise.

final class SessionChatRepositoryProvider
    extends $FunctionalProvider<ChatRepository, ChatRepository, ChatRepository>
    with $Provider<ChatRepository> {
  /// Resolves the [ChatRepository] for a specific [sessionId].
  ///
  /// Returns the WS-backed repository when connected, [EmptyChatRepository]
  /// otherwise.
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
    r'2d51cb57ead11967b5c6446b04badd16c685b2de';

/// Resolves the [ChatRepository] for a specific [sessionId].
///
/// Returns the WS-backed repository when connected, [EmptyChatRepository]
/// otherwise.

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
  /// Returns the WS-backed repository when connected, [EmptyChatRepository]
  /// otherwise.

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

String _$chatMessagesHash() => r'b1c3120386006adb92bee93151a958abb8089689';

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

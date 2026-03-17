// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'agent_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Tracks agents from lifecycle events.

@ProviderFor(Agents)
final agentsProvider = AgentsFamily._();

/// Tracks agents from lifecycle events.
final class AgentsProvider extends $NotifierProvider<Agents, List<AgentInfo>> {
  /// Tracks agents from lifecycle events.
  AgentsProvider._({
    required AgentsFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'agentsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$agentsHash();

  @override
  String toString() {
    return r'agentsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  Agents create() => Agents();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<AgentInfo> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<AgentInfo>>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is AgentsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$agentsHash() => r'22491ffb55c6280c206d22499434acbe6e097c03';

/// Tracks agents from lifecycle events.

final class AgentsFamily extends $Family
    with
        $ClassFamilyOverride<
          Agents,
          List<AgentInfo>,
          List<AgentInfo>,
          List<AgentInfo>,
          String
        > {
  AgentsFamily._()
    : super(
        retry: null,
        name: r'agentsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Tracks agents from lifecycle events.

  AgentsProvider call(String sessionId) =>
      AgentsProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'agentsProvider';
}

/// Tracks agents from lifecycle events.

abstract class _$Agents extends $Notifier<List<AgentInfo>> {
  late final _$args = ref.$arg as String;
  String get sessionId => _$args;

  List<AgentInfo> build(String sessionId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<AgentInfo>, List<AgentInfo>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<AgentInfo>, List<AgentInfo>>,
              List<AgentInfo>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

/// Selected agent ID. null = supervisor view.

@ProviderFor(SelectedAgent)
final selectedAgentProvider = SelectedAgentFamily._();

/// Selected agent ID. null = supervisor view.
final class SelectedAgentProvider
    extends $NotifierProvider<SelectedAgent, String?> {
  /// Selected agent ID. null = supervisor view.
  SelectedAgentProvider._({
    required SelectedAgentFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'selectedAgentProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$selectedAgentHash();

  @override
  String toString() {
    return r'selectedAgentProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  SelectedAgent create() => SelectedAgent();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String?>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is SelectedAgentProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$selectedAgentHash() => r'153afb7d2a0cc69c32d04cfb648b21f861d07e01';

/// Selected agent ID. null = supervisor view.

final class SelectedAgentFamily extends $Family
    with
        $ClassFamilyOverride<SelectedAgent, String?, String?, String?, String> {
  SelectedAgentFamily._()
    : super(
        retry: null,
        name: r'selectedAgentProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Selected agent ID. null = supervisor view.

  SelectedAgentProvider call(String sessionId) =>
      SelectedAgentProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'selectedAgentProvider';
}

/// Selected agent ID. null = supervisor view.

abstract class _$SelectedAgent extends $Notifier<String?> {
  late final _$args = ref.$arg as String;
  String get sessionId => _$args;

  String? build(String sessionId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<String?, String?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<String?, String?>,
              String?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

/// Filtered messages by selected agent.

@ProviderFor(filteredChatMessages)
final filteredChatMessagesProvider = FilteredChatMessagesFamily._();

/// Filtered messages by selected agent.

final class FilteredChatMessagesProvider
    extends
        $FunctionalProvider<
          List<ChatMessage>,
          List<ChatMessage>,
          List<ChatMessage>
        >
    with $Provider<List<ChatMessage>> {
  /// Filtered messages by selected agent.
  FilteredChatMessagesProvider._({
    required FilteredChatMessagesFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'filteredChatMessagesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$filteredChatMessagesHash();

  @override
  String toString() {
    return r'filteredChatMessagesProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<List<ChatMessage>> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  List<ChatMessage> create(Ref ref) {
    final argument = this.argument as String;
    return filteredChatMessages(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<ChatMessage> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<ChatMessage>>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is FilteredChatMessagesProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$filteredChatMessagesHash() =>
    r'4201884de2270cb0e0903093cd4ed5742b120253';

/// Filtered messages by selected agent.

final class FilteredChatMessagesFamily extends $Family
    with $FunctionalFamilyOverride<List<ChatMessage>, String> {
  FilteredChatMessagesFamily._()
    : super(
        retry: null,
        name: r'filteredChatMessagesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Filtered messages by selected agent.

  FilteredChatMessagesProvider call(String sessionId) =>
      FilteredChatMessagesProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'filteredChatMessagesProvider';
}

/// Whether multi-agent mode is active.

@ProviderFor(isMultiAgent)
final isMultiAgentProvider = IsMultiAgentFamily._();

/// Whether multi-agent mode is active.

final class IsMultiAgentProvider extends $FunctionalProvider<bool, bool, bool>
    with $Provider<bool> {
  /// Whether multi-agent mode is active.
  IsMultiAgentProvider._({
    required IsMultiAgentFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'isMultiAgentProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$isMultiAgentHash();

  @override
  String toString() {
    return r'isMultiAgentProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<bool> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  bool create(Ref ref) {
    final argument = this.argument as String;
    return isMultiAgent(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is IsMultiAgentProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$isMultiAgentHash() => r'b0fbe89ced2ed1798c567c7a885b692f1a531d90';

/// Whether multi-agent mode is active.

final class IsMultiAgentFamily extends $Family
    with $FunctionalFamilyOverride<bool, String> {
  IsMultiAgentFamily._()
    : super(
        retry: null,
        name: r'isMultiAgentProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Whether multi-agent mode is active.

  IsMultiAgentProvider call(String sessionId) =>
      IsMultiAgentProvider._(argument: sessionId, from: this);

  @override
  String toString() => r'isMultiAgentProvider';
}

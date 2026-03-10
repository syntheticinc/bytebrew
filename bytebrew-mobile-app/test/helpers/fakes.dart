import 'dart:async';

import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/core/domain/auth_tokens.dart';
import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/auth/domain/auth_repository.dart';
import 'package:bytebrew_mobile/features/auth/infrastructure/token_storage.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/features/sessions/application/sessions_provider.dart';
import 'package:bytebrew_mobile/features/settings/domain/settings_repository.dart';

/// Fake [SettingsRepository] that returns an empty server list and does
/// nothing on removal.
///
/// Prevents tests from reaching [sharedPreferencesProvider] (which throws
/// [UnimplementedError] when not overridden).
class FakeSettingsRepository implements SettingsRepository {
  FakeSettingsRepository([this._servers = const []]);

  final List<Server> _servers;

  @override
  List<Server> getServers() => _servers;

  @override
  Future<List<Server>> getServersWithKeys() async => _servers;

  @override
  Future<void> addServer(Server server) async {}

  @override
  Future<void> removeServer(String id) async {}
}

/// Fake [TokenStorage] that stores nothing (no SharedPreferences needed).
///
/// Useful for tests that override [tokenStorageProvider] without requiring
/// platform channel setup.
class FakeTokenStorage extends TokenStorage {
  @override
  Future<void> saveTokens(AuthTokens tokens) async {}

  @override
  Future<AuthTokens?> getTokens() async => null;

  @override
  Future<void> clearTokens() async {}

  @override
  Future<bool> hasTokens() async => false;
}

/// Fake [AuthRepository] with configurable login/register behaviour.
///
/// By default, returns dummy tokens. Set [shouldSucceed] to false to make
/// login/register throw an exception with [errorMessage].
class FakeAuthRepository implements AuthRepository {
  FakeAuthRepository({
    this.shouldSucceed = true,
    this.loginResult = const AuthTokens(accessToken: 'a', refreshToken: 'r'),
    this.errorMessage = 'Auth failed',
  });

  final bool shouldSucceed;
  final AuthTokens loginResult;
  final String errorMessage;

  @override
  Future<AuthTokens> login(String email, String password) async {
    if (!shouldSucceed) {
      throw Exception(errorMessage);
    }
    return loginResult;
  }

  @override
  Future<AuthTokens> register(String email, String password) async {
    if (!shouldSucceed) {
      throw Exception(errorMessage);
    }
    return loginResult;
  }

  @override
  Future<void> logout() async {}
}

/// Fake [Sessions] notifier that immediately resolves with a given list.
///
/// This was duplicated identically in chat_screen_test.dart and
/// sessions_screen_test.dart.
class FakeSessionsNotifier extends Sessions {
  FakeSessionsNotifier(this._sessions);

  final List<Session> _sessions;

  @override
  FutureOr<List<Session>> build() => _sessions;

  @override
  Future<void> refresh() async {}
}

/// Fake [ChatRepository] that returns a fixed message list.
///
/// Suitable for simple read-only tests. For stream-based tests, use
/// [StreamableFakeChatRepository].
class FakeChatRepository implements ChatRepository {
  FakeChatRepository(this._messages);

  final List<ChatMessage> _messages;

  @override
  Future<List<ChatMessage>> getMessages(String sessionId) async => _messages;

  @override
  Future<void> sendMessage(String sessionId, String text) async {}

  @override
  Future<void> answerAskUser(
    String sessionId,
    String askUserId,
    String answer,
  ) async {}

  @override
  Future<void> cancel(String sessionId) async {}

  @override
  Stream<List<ChatMessage>>? watchMessages() => null;

  @override
  Stream<List<AgentInfo>>? watchAgents() => null;

  @override
  Stream<bool>? watchProcessing() => null;
}

/// Extended fake [ChatRepository] with a [StreamController] for real-time
/// message testing.
///
/// Use [emitMessages] to push new message lists into the stream and
/// inspect [sentMessages] to verify what was sent.
class StreamableFakeChatRepository implements ChatRepository {
  StreamableFakeChatRepository({List<ChatMessage>? initialMessages})
    : _messages = initialMessages ?? [];

  final List<ChatMessage> _messages;
  final _controller = StreamController<List<ChatMessage>>.broadcast();

  /// Messages recorded by [sendMessage] calls.
  final List<String> sentMessages = [];

  /// Pushes a new message list into the [watchMessages] stream.
  void emitMessages(List<ChatMessage> messages) {
    _controller.add(messages);
  }

  /// Closes the underlying stream controller. Call in tearDown.
  void dispose() {
    _controller.close();
  }

  @override
  Future<List<ChatMessage>> getMessages(String sessionId) async => _messages;

  @override
  Future<void> sendMessage(String sessionId, String text) async {
    sentMessages.add(text);
  }

  @override
  Future<void> answerAskUser(
    String sessionId,
    String askUserId,
    String answer,
  ) async {}

  @override
  Future<void> cancel(String sessionId) async {}

  @override
  Stream<List<ChatMessage>> watchMessages() => _controller.stream;

  @override
  Stream<List<AgentInfo>>? watchAgents() => null;

  @override
  Stream<bool>? watchProcessing() => null;
}

/// Extended [StreamableFakeChatRepository] with agent stream support.
///
/// Use [emitAgents] to push agent lists into the [watchAgents] stream.
/// Useful for testing multi-agent UI components (e.g. AgentSelectorBar).
class StreamableFakeAgentChatRepository extends StreamableFakeChatRepository {
  StreamableFakeAgentChatRepository({super.initialMessages});

  final _agentsController = StreamController<List<AgentInfo>>.broadcast();

  /// Pushes a new agent list into the [watchAgents] stream.
  void emitAgents(List<AgentInfo> agents) {
    _agentsController.add(agents);
  }

  @override
  Stream<List<AgentInfo>> watchAgents() => _agentsController.stream;

  @override
  void dispose() {
    _agentsController.close();
    super.dispose();
  }
}

/// Fake [WsConnectionManager] for widget tests.
///
/// Returns a fixed set of connections without establishing real WS channels.
class FakeConnectionManager extends WsConnectionManager {
  FakeConnectionManager({Map<String, WsServerConnection>? initialConnections})
    : _fakeConnections = initialConnections ?? {};

  final Map<String, WsServerConnection> _fakeConnections;

  /// Adds a connection for testing.
  void addFakeConnection(String serverId, WsServerConnection connection) {
    _fakeConnections[serverId] = connection;
  }

  @override
  Map<String, WsServerConnection> get connections =>
      Map.unmodifiable(_fakeConnections);

  @override
  Iterable<WsServerConnection> get activeConnections => _fakeConnections.values
      .where((c) => c.status == WsConnectionStatus.connected);

  @override
  WsServerConnection? getConnection(String serverId) =>
      _fakeConnections[serverId];
}

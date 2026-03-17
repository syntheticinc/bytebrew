/// E2E transport diagnostic test — progressively adds real app layers
/// until the bug is reproduced.
///
/// Prerequisites:
///   - CLI running (with bridge enabled) + gRPC server
///   - At least one paired device (/mobile pair)
///
/// Run:
///   flutter test test/integration/transport_chain_test.dart \
///     --dart-define=SERVER_ID=... --dart-define=DEVICE_ID=... \
///     --dart-define=DEVICE_TOKEN=... --dart-define=SHARED_SECRET_HEX=...
library;

import 'dart:typed_data';

import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/chat/application/chat_provider.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------
const _bridgeUrl = String.fromEnvironment(
  'BRIDGE_URL',
  defaultValue: 'wss://bridge.bytebrew.ai',
);
const _serverId = String.fromEnvironment('SERVER_ID', defaultValue: '');
const _deviceId = String.fromEnvironment('DEVICE_ID', defaultValue: '');
const _deviceToken = String.fromEnvironment('DEVICE_TOKEN', defaultValue: '');
const _sharedSecretHex = String.fromEnvironment(
  'SHARED_SECRET_HEX',
  defaultValue: '',
);

Uint8List _hexDecode(String hex) {
  final result = Uint8List(hex.length ~/ 2);
  for (var i = 0; i < hex.length; i += 2) {
    result[i ~/ 2] = int.parse(hex.substring(i, i + 2), radix: 16);
  }
  return result;
}

void _log(String msg) {
  final now = DateTime.now();
  final ts = '${now.hour.toString().padLeft(2, '0')}:'
      '${now.minute.toString().padLeft(2, '0')}:'
      '${now.second.toString().padLeft(2, '0')}.'
      '${now.millisecond.toString().padLeft(3, '0')}';
  // ignore: avoid_print
  print('[$ts] $msg');
}

Server get _server => Server(
  id: _serverId,
  name: 'test-cli',
  bridgeUrl: _bridgeUrl,
  isOnline: true,
  pairedAt: DateTime.now(),
  deviceToken: _deviceToken,
  deviceId: _deviceId,
  sharedSecret:
      _sharedSecretHex.isNotEmpty ? _hexDecode(_sharedSecretHex) : null,
);

int _countAssistantMessages(List<ChatMessage> messages) {
  return messages
      .where(
        (m) => m.type == ChatMessageType.agentMessage && m.content.isNotEmpty,
      )
      .length;
}

void main() {
  final skip = _serverId.isEmpty || _deviceId.isEmpty || _deviceToken.isEmpty
      ? 'Set SERVER_ID, DEVICE_ID, DEVICE_TOKEN, SHARED_SECRET_HEX env vars'
      : null;

  // =========================================================================
  // Level 0: Idle delay — connect, wait 90s, THEN subscribe + send.
  // Reproduces the real app scenario where connection sits idle before chat.
  // =========================================================================
  test(
    'L0: idle delay before subscribe + send',
    skip: skip,
    timeout: const Timeout(Duration(minutes: 5)),
    () async {
      _log('=== L0: Idle Delay Test ===');

      final manager = WsConnectionManager();
      await manager.connectToServer(_server);

      final conn = manager.getConnection(_serverId);
      expect(conn?.status, WsConnectionStatus.connected);
      _log('Connected. Idling for 90 seconds...');

      // Idle — only WS pings should be flowing
      await Future<void>.delayed(const Duration(seconds: 90));

      _log('Idle complete. Checking connection...');
      final connAfter = manager.getConnection(_serverId);
      _log('Connection status after idle: ${connAfter?.status}');
      expect(connAfter?.status, WsConnectionStatus.connected,
          reason: 'Connection lost during idle');

      // Now do what the real app does: list sessions, subscribe, send
      final sessions = await connAfter!.client.listSessions(
        deviceToken: _deviceToken,
      );
      expect(sessions.sessions, isNotEmpty, reason: 'No sessions on CLI');
      final sessionId = sessions.sessions.first.sessionId;
      _log('Session after idle: $sessionId');

      final repo = WsChatRepository(
        connectionManager: manager,
        serverId: _serverId,
        sessionId: sessionId,
      );
      await repo.subscribe();
      await Future<void>.delayed(const Duration(milliseconds: 500));

      _log('Sending message after idle...');
      int count = 0;
      final sub = repo.watchMessages().listen((msgs) {
        final c = _countAssistantMessages(msgs);
        if (c > count) {
          count = c;
          _log('  Assistant messages: $count');
        }
      });

      await repo.sendMessage(sessionId, 'Ответь одним словом: 10+5?');

      final deadline = DateTime.now().add(const Duration(seconds: 60));
      while (count < 1 && DateTime.now().isBefore(deadline)) {
        await Future<void>.delayed(const Duration(milliseconds: 300));
      }

      _log('Result: $count assistant messages');
      await sub.cancel();
      repo.dispose();
      await manager.disconnectAll();

      expect(count, greaterThanOrEqualTo(1),
          reason: 'No response after 90s idle');
    },
  );

  // =========================================================================
  // Level 1: Real WsConnectionManager.connectToServer()
  // =========================================================================
  test(
    'L1: real WsConnectionManager + WsChatRepository',
    skip: skip,
    timeout: const Timeout(Duration(minutes: 5)),
    () async {
      _log('=== L1: Real WsConnectionManager ===');

      final manager = WsConnectionManager();
      await manager.connectToServer(_server);

      final conn = manager.getConnection(_serverId);
      _log('Connection status: ${conn?.status}');
      expect(conn?.status, WsConnectionStatus.connected);

      final sessions = await conn!.client.listSessions(
        deviceToken: _deviceToken,
      );
      expect(sessions.sessions, isNotEmpty, reason: 'No sessions on CLI');
      final sessionId = sessions.sessions.first.sessionId;
      _log('Session: $sessionId');

      final repo = WsChatRepository(
        connectionManager: manager,
        serverId: _serverId,
        sessionId: sessionId,
      );
      await repo.subscribe();
      await Future<void>.delayed(const Duration(milliseconds: 500));

      // Send and wait
      _log('Sending message...');
      int count = 0;
      final sub = repo.watchMessages().listen((msgs) {
        final c = _countAssistantMessages(msgs);
        if (c > count) {
          count = c;
          _log('  Assistant messages: $count');
        }
      });

      await repo.sendMessage(sessionId, 'Ответь одним словом: 2+3?');

      final deadline = DateTime.now().add(const Duration(seconds: 60));
      while (count < 1 && DateTime.now().isBefore(deadline)) {
        await Future<void>.delayed(const Duration(milliseconds: 300));
      }

      _log('Result: $count assistant messages');
      await sub.cancel();
      repo.dispose();
      await manager.disconnectAll();

      expect(count, greaterThanOrEqualTo(1));
    },
  );

  // =========================================================================
  // Level 2: Real WsConnectionManager + dispose/re-subscribe
  // =========================================================================
  test(
    'L2: real WsConnectionManager + dispose + re-subscribe',
    skip: skip,
    timeout: const Timeout(Duration(minutes: 5)),
    () async {
      _log('=== L2: Dispose + Re-subscribe ===');

      final manager = WsConnectionManager();
      await manager.connectToServer(_server);

      final conn = manager.getConnection(_serverId)!;
      final sessions = await conn.client.listSessions(
        deviceToken: _deviceToken,
      );
      final sessionId = sessions.sessions.first.sessionId;

      // Phase 1
      _log('[Phase 1] Create + subscribe + send');
      var repo = WsChatRepository(
        connectionManager: manager,
        serverId: _serverId,
        sessionId: sessionId,
      );
      await repo.subscribe();
      await Future<void>.delayed(const Duration(milliseconds: 500));

      int count = 0;
      var sub = repo.watchMessages().listen((msgs) {
        count = _countAssistantMessages(msgs);
      });
      await repo.sendMessage(sessionId, 'Ответь одним словом: 1+1?');

      var dl = DateTime.now().add(const Duration(seconds: 60));
      while (count < 1 && DateTime.now().isBefore(dl)) {
        await Future<void>.delayed(const Duration(milliseconds: 300));
      }
      _log('[Phase 1] Result: $count');
      expect(count, greaterThanOrEqualTo(1), reason: 'Phase 1');
      await sub.cancel();
      repo.dispose();

      // Phase 2: wait
      await Future<void>.delayed(const Duration(seconds: 2));

      // Phase 3
      _log('[Phase 3] Re-create + subscribe + send');
      repo = WsChatRepository(
        connectionManager: manager,
        serverId: _serverId,
        sessionId: sessionId,
      );
      await repo.subscribe();
      await Future<void>.delayed(const Duration(milliseconds: 500));

      count = 0;
      sub = repo.watchMessages().listen((msgs) {
        count = _countAssistantMessages(msgs);
      });
      await repo.sendMessage(sessionId, 'Ответь одним словом: 3+3?');

      dl = DateTime.now().add(const Duration(seconds: 60));
      while (count < 1 && DateTime.now().isBefore(dl)) {
        await Future<void>.delayed(const Duration(milliseconds: 300));
      }
      _log('[Phase 3] Result: $count');
      await sub.cancel();
      repo.dispose();
      await manager.disconnectAll();

      expect(count, greaterThanOrEqualTo(1), reason: 'Phase 3');
    },
  );

  // =========================================================================
  // Level 3: Full Riverpod stack (exactly like the real app)
  //   connectionManagerProvider → sessionsProvider → sessionChatRepositoryProvider
  //   → ChatMessages notifier
  // =========================================================================
  test(
    'L3: full Riverpod stack (ChatMessages notifier)',
    skip: skip,
    timeout: const Timeout(Duration(minutes: 5)),
    () async {
      _log('=== L3: Full Riverpod Stack ===');

      // 1. Connect
      final manager = WsConnectionManager();
      await manager.connectToServer(_server);

      final conn = manager.getConnection(_serverId)!;
      final sessionsResult = await conn.client.listSessions(
        deviceToken: _deviceToken,
      );
      expect(sessionsResult.sessions, isNotEmpty);
      final sessionId = sessionsResult.sessions.first.sessionId;
      _log('Session: $sessionId');

      // 2. Build ProviderContainer with overrides matching the real app
      final container = ProviderContainer(
        overrides: [
          connectionManagerProvider.overrideWithValue(manager),
          // Override sessionChatRepository directly — it uses sessionsProvider
          // internally, so we bypass that by providing the repo ourselves.
          sessionChatRepositoryProvider(sessionId).overrideWith((ref) {
            final repo = WsChatRepository(
              connectionManager: manager,
              serverId: _serverId,
              sessionId: sessionId,
            );
            await repo.subscribe();
            ref.onDispose(repo.dispose);
            return repo;
          }),
        ],
      );
      addTearDown(() async {
        container.dispose();
        await manager.disconnectAll();
      });

      // 3. Listen to ChatMessages — keeps the auto-dispose provider alive
      // (in the real app, ChatScreen does ref.watch which keeps it alive)
      _log('Listening to chatMessagesProvider...');
      final subscription = container.listen(
        chatMessagesProvider(sessionId),
        (prev, next) {
          _log('  [Riverpod] state changed: ${next.value?.length ?? 0} messages');
        },
      );
      addTearDown(subscription.close);

      final notifier = container.read(
        chatMessagesProvider(sessionId).notifier,
      );

      await Future<void>.delayed(const Duration(seconds: 1));

      // 4. Send 3 messages via ChatMessages.sendMessage() (like the UI)
      final questions = [
        'Ответь одним словом: какой сегодня день недели?',
        'Ответь одним словом: сколько будет 2+2?',
        'Ответь одним словом: какой сейчас месяц?',
      ];

      for (var i = 0; i < questions.length; i++) {
        _log('');
        _log('[MSG ${i + 1}/3] Sending: "${questions[i]}"');

        await notifier.sendMessage(questions[i]);

        final target = i + 1;
        final deadline = DateTime.now().add(const Duration(seconds: 60));
        int assistantCount = 0;

        while (DateTime.now().isBefore(deadline)) {
          final state = container.read(chatMessagesProvider(sessionId));
          final messages = state.value ?? [];
          assistantCount = _countAssistantMessages(messages);
          if (assistantCount >= target) break;
          await Future<void>.delayed(const Duration(milliseconds: 300));
        }

        _log('[MSG ${i + 1}/3] assistantCount=$assistantCount (target=$target)');

        if (assistantCount < target) {
          // Dump all messages for debugging
          final state = container.read(chatMessagesProvider(sessionId));
          final msgs = state.value ?? [];
          _log('ALL MESSAGES (${msgs.length}):');
          for (final m in msgs) {
            _log(
              '  ${m.type.name} id=${m.id} '
              'content="${m.content.length > 60 ? '${m.content.substring(0, 60)}...' : m.content}"',
            );
          }
          fail(
            'MSG ${i + 1}/3: expected $target assistant messages, '
            'got $assistantCount',
          );
        }

        if (i < questions.length - 1) {
          await Future<void>.delayed(const Duration(seconds: 2));
        }
      }

      _log('');
      _log('=== L3 PASSED: 3/3 messages ===');
    },
  );

  // =========================================================================
  // Level 4: Simulate internal WsConnection reconnect WITHOUT manager update.
  // This reproduces the app-only bug where the socket reconnects, but
  // WsConnectionManager never transitions back to "connected", so
  // WsChatRepository does not re-subscribe and sends time out.
  // =========================================================================
  test(
    'L4: internal reconnect leaves manager disconnected (repro)',
    skip: skip,
    timeout: const Timeout(Duration(minutes: 5)),
    () async {
      _log('=== L4: Internal Reconnect Without Manager Update ===');

      final manager = WsConnectionManager();
      await manager.connectToServer(_server);

      final serverConn = manager.getConnection(_serverId);
      expect(serverConn?.status, WsConnectionStatus.connected);

      final sessions = await serverConn!.client.listSessions(
        deviceToken: _deviceToken,
      );
      expect(sessions.sessions, isNotEmpty, reason: 'No sessions on CLI');
      final sessionId = sessions.sessions.first.sessionId;

      final repo = WsChatRepository(
        connectionManager: manager,
        serverId: _serverId,
        sessionId: sessionId,
      );
      await repo.subscribe();
      await Future<void>.delayed(const Duration(milliseconds: 500));

      // Phase 1: verify normal behavior
      int count = 0;
      final sub = repo.watchMessages().listen((msgs) {
        count = _countAssistantMessages(msgs);
      });
      await repo.sendMessage(sessionId, 'Ответь одним словом: 4+4?');

      var deadline = DateTime.now().add(const Duration(seconds: 60));
      while (count < 1 && DateTime.now().isBefore(deadline)) {
        await Future<void>.delayed(const Duration(milliseconds: 300));
      }
      expect(count, greaterThanOrEqualTo(1), reason: 'Phase 1');

      // Phase 2: simulate internal reconnect
      _log('[Phase 2] Disconnecting WsConnection directly...');
      await serverConn.connection.disconnect();

      // Directly reconnect the underlying WsConnection (bypasses manager).
      _log('[Phase 2] Reconnecting WsConnection directly...');
      await serverConn.connection.connect();

      // After fix: manager mirrors WsConnection status 1:1, so it should
      // transition to connected as soon as WsConnection emits connected.
      _log(
        '[Phase 2] wsConnection.status=${serverConn.connection.status} '
        'managerStatus=${serverConn.status}',
      );
      expect(
        serverConn.connection.status,
        WsConnectionStatus.connected,
        reason: 'Underlying WsConnection should be connected',
      );

      // Manager mirrors status synchronously via listener — should already be connected.
      // Give a small grace period for the listener to fire.
      await Future<void>.delayed(const Duration(milliseconds: 500));

      _log('[Phase 2] After mirror: managerStatus=${serverConn.status}');
      expect(
        serverConn.status,
        WsConnectionStatus.connected,
        reason: 'Manager should mirror connected status from WsConnection',
      );

      // Phase 3: send after reconnect — should work now.
      _log('[Phase 3] Sending after internal reconnect...');
      _log(
        '[Phase 3] repo.isSubscribed=${repo.isSubscribed} '
        'wsConnection.isStale=${serverConn.connection.isStale}',
      );
      final beforeCount = count;
      await repo.sendMessage(sessionId, 'Ответь одним словом: 5+5?');

      deadline = DateTime.now().add(const Duration(seconds: 60));
      while (count == beforeCount && DateTime.now().isBefore(deadline)) {
        await Future<void>.delayed(const Duration(milliseconds: 300));
      }

      _log(
        '[Phase 3] assistantCount=$count (before=$beforeCount) '
        'managerStatus=${serverConn.status} wsStatus=${serverConn.connection.status}',
      );

      await sub.cancel();
      repo.dispose();
      await manager.disconnectAll();

      expect(
        count,
        greaterThan(beforeCount),
        reason:
            'New assistant messages expected after manager syncs to connected',
      );
    },
  );
}

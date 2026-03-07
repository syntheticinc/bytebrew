/// Flutter E2E integration tests against a real backend chain.
///
/// Tests the full path: Flutter app -> real WS -> real Bridge -> real CLI ->
/// real Server (MockLLM). Uses [BackendFixture] to start and manage the
/// backend processes.
///
/// Run with:
/// ```bash
/// cd bytebrew-mobile-app
/// flutter test integration_test/backend_e2e_test.dart
/// ```
library;

import 'dart:async';

import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers/backend_fixture.dart';
import 'helpers/test_app.dart';

// ---------------------------------------------------------------------------
// Shared state
// ---------------------------------------------------------------------------

/// Credentials obtained from the pairing flow in FE2E-01.
///
/// Populated by the first test and reused by all subsequent tests to avoid
/// consuming additional pairing tokens.
late String _pairedDeviceToken;
late String _pairedDeviceId;
late String _pairedServerId;
late String _pairedServerName;

/// Whether the shared pairing has been completed.
bool _isPaired = false;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Creates a fresh [WsConnection] to the backend Bridge.
WsConnection _createConnection(BackendFixture backend, {String? deviceId}) {
  return WsConnection(
    bridgeUrl: backend.bridgeUrl,
    serverId: backend.serverId,
    deviceId: deviceId ?? 'e2e-${DateTime.now().millisecondsSinceEpoch}',
  );
}

/// Creates a [WsBridgeClient] on top of an existing [connection].
WsBridgeClient _createClient(WsConnection connection) {
  return WsBridgeClient(connection: connection, deviceId: connection.deviceId);
}

/// Builds a [Server] from the paired credentials and the [backend] fixture.
Server _buildPairedServer(BackendFixture backend) {
  return Server(
    id: _pairedServerId,
    name: _pairedServerName,
    bridgeUrl: backend.bridgeUrl,
    isOnline: true,
    pairedAt: DateTime.now(),
    deviceToken: _pairedDeviceToken,
    deviceId: _pairedDeviceId,
  );
}

/// Waits for [condition] to return `true`, polling at [interval].
///
/// Throws [TimeoutException] if the condition is not met within [timeout].
Future<void> _waitForCondition(
  bool Function() condition, {
  Duration timeout = const Duration(seconds: 10),
  Duration interval = const Duration(milliseconds: 100),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(deadline)) {
    if (condition()) return;
    await Future<void>.delayed(interval);
  }
  throw TimeoutException('Condition not met within $timeout', timeout);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  initializeBinding();

  late BackendFixture backend;

  setUpAll(() async {
    // Build all backend binaries (Bridge, CLI, Server) if needed.
    await BackendFixture.buildAll();

    // Start the backend processes with the echo (MockLLM) scenario.
    backend = BackendFixture();
    await backend.start(scenario: 'echo');
  });

  tearDownAll(() async {
    await backend.stop();
  });

  // -------------------------------------------------------------------------
  // FE2E-01: Real WS pairing flow
  // -------------------------------------------------------------------------

  testWidgets('FE2E-01: Real WS pairing via WsBridgeClient', (tester) async {
    await tester.runAsync(() async {
      final connection = _createConnection(
        backend,
        deviceId: 'pair-device-${DateTime.now().millisecondsSinceEpoch}',
      );

      try {
        await connection.connect();
        expect(
          connection.status,
          WsConnectionStatus.connected,
          reason: 'WS connection to Bridge should be established',
        );

        final client = _createClient(connection);

        try {
          final result = await client.pair(
            token: backend.pairingToken,
            deviceName: 'Flutter E2E Test Device',
          );

          // Verify pairing response.
          expect(
            result.deviceToken,
            isNotEmpty,
            reason: 'Pairing should return a device token',
          );
          expect(
            result.deviceId,
            isNotEmpty,
            reason: 'Pairing should return a device ID',
          );

          // Store shared credentials for subsequent tests.
          _pairedDeviceToken = result.deviceToken;
          _pairedDeviceId = result.deviceId;
          _pairedServerId = result.serverId.isNotEmpty
              ? result.serverId
              : backend.serverId;
          _pairedServerName = result.serverName.isNotEmpty
              ? result.serverName
              : 'CLI Server';
          _isPaired = true;
        } finally {
          await client.dispose();
        }
      } finally {
        await connection.dispose();
      }
    });

    // Guard: ensure pairing actually succeeded for subsequent tests.
    expect(_isPaired, isTrue, reason: 'Pairing must succeed for other tests');
  });

  // -------------------------------------------------------------------------
  // FE2E-02: List sessions via real WS
  // -------------------------------------------------------------------------

  testWidgets('FE2E-02: List sessions via real WS', (tester) async {
    expect(_isPaired, isTrue, reason: 'FE2E-01 must pass first');

    await tester.runAsync(() async {
      final connection = _createConnection(backend, deviceId: _pairedDeviceId);

      try {
        await connection.connect();
        final client = _createClient(connection);

        try {
          final result = await client.listSessions(
            deviceToken: _pairedDeviceToken,
          );

          expect(
            result.sessions,
            isNotEmpty,
            reason: 'Backend should have at least one session',
          );

          // Verify the known session exists.
          final found = result.sessions
              .where((s) => s.sessionId == backend.sessionId)
              .toList();
          expect(
            found,
            isNotEmpty,
            reason:
                'Session ${backend.sessionId} should be in the session list',
          );
        } finally {
          await client.dispose();
        }
      } finally {
        await connection.dispose();
      }
    });
  });

  // -------------------------------------------------------------------------
  // FE2E-03: Send message + receive echo response
  // -------------------------------------------------------------------------

  testWidgets('FE2E-03: Send message, receive echo response via WS', (
    tester,
  ) async {
    expect(_isPaired, isTrue, reason: 'FE2E-01 must pass first');

    await tester.runAsync(() async {
      final connection = _createConnection(backend, deviceId: _pairedDeviceId);

      try {
        await connection.connect();
        final client = _createClient(connection);

        try {
          // Subscribe to session events.
          final events = <SessionEvent>[];
          final eventStream = client.subscribeSession(
            deviceToken: _pairedDeviceToken,
            sessionId: backend.sessionId,
          );
          final subscription = eventStream.listen(events.add);

          try {
            // Small delay to let the subscription register on the server side.
            await Future<void>.delayed(const Duration(milliseconds: 200));

            // Send a new task.
            final sendResult = await client.sendNewTask(
              deviceToken: _pairedDeviceToken,
              sessionId: backend.sessionId,
              task: 'hello world',
            );
            expect(
              sendResult.success,
              isTrue,
              reason: 'sendNewTask should succeed',
            );

            // Wait for an agent message event (MessageCompleted).
            await _waitForCondition(
              () => events.any(
                (e) =>
                    e.type == SessionEventType.agentMessage &&
                    e.payload is AgentMessagePayload &&
                    (e.payload! as AgentMessagePayload).isComplete,
              ),
              timeout: const Duration(seconds: 30),
            );

            // Verify we received at least one agent message event.
            final agentMessages = events
                .where((e) => e.type == SessionEventType.agentMessage)
                .toList();
            expect(
              agentMessages,
              isNotEmpty,
              reason: 'Should receive at least one agent message event',
            );

            // The completed message should have content.
            final completed = agentMessages
                .where((e) => (e.payload! as AgentMessagePayload).isComplete)
                .first;
            final completedPayload = completed.payload! as AgentMessagePayload;
            expect(
              completedPayload.content,
              isNotEmpty,
              reason: 'Completed agent message should have content',
            );
          } finally {
            await subscription.cancel();
          }
        } finally {
          await client.dispose();
        }
      } finally {
        await connection.dispose();
      }
    });
  });

  // -------------------------------------------------------------------------
  // FE2E-04: WsConnectionManager connects to real server
  // -------------------------------------------------------------------------

  testWidgets('FE2E-04: WsConnectionManager connects to real server', (
    tester,
  ) async {
    expect(_isPaired, isTrue, reason: 'FE2E-01 must pass first');

    await tester.runAsync(() async {
      final server = _buildPairedServer(backend);
      final manager = WsConnectionManager();

      try {
        await manager.connectToServer(server);

        final conn = manager.getConnection(_pairedServerId);
        expect(
          conn,
          isNotNull,
          reason: 'Connection should be stored in the manager',
        );
        expect(
          conn!.status,
          WsConnectionStatus.connected,
          reason:
              'Connection should be in connected state after connectToServer',
        );

        // Verify the connection is in activeConnections.
        expect(
          manager.activeConnections.any((c) => c.server.id == _pairedServerId),
          isTrue,
          reason: 'Server should appear in activeConnections',
        );
      } finally {
        await manager.disconnectAll();
        manager.dispose();
      }
    });
  });

  // -------------------------------------------------------------------------
  // FE2E-05: WsChatRepository send + receive (echo scenario)
  // -------------------------------------------------------------------------

  testWidgets('FE2E-05: WsChatRepository send and receive via real WS', (
    tester,
  ) async {
    expect(_isPaired, isTrue, reason: 'FE2E-01 must pass first');

    await tester.runAsync(() async {
      // Connect via WsConnectionManager.
      final server = _buildPairedServer(backend);
      final manager = WsConnectionManager();

      try {
        await manager.connectToServer(server);

        final conn = manager.getConnection(_pairedServerId);
        expect(conn, isNotNull, reason: 'Should be connected');
        expect(conn!.status, WsConnectionStatus.connected);

        // Create the real WsChatRepository.
        final chatRepo = WsChatRepository(
          connectionManager: manager,
          serverId: _pairedServerId,
          sessionId: backend.sessionId,
        );
        chatRepo.subscribe();

        try {
          // Listen for message emissions.
          final messageSnapshots = <List<ChatMessage>>[];
          final sub = chatRepo.watchMessages().listen(messageSnapshots.add);

          try {
            // Send a message.
            await chatRepo.sendMessage(backend.sessionId, 'hello echo');

            // The user message should appear immediately (optimistic update).
            await _waitForCondition(
              () => messageSnapshots.any(
                (list) => list.any(
                  (m) =>
                      m.type == ChatMessageType.userMessage &&
                      m.content == 'hello echo',
                ),
              ),
              timeout: const Duration(seconds: 5),
            );

            // Wait for an agent response message via the event subscription.
            await _waitForCondition(
              () => messageSnapshots.any(
                (list) =>
                    list.any((m) => m.type == ChatMessageType.agentMessage),
              ),
              timeout: const Duration(seconds: 30),
            );

            // Verify the final message list contains both user and agent
            // messages.
            final lastSnapshot = messageSnapshots.last;

            final hasUserMessage = lastSnapshot.any(
              (m) =>
                  m.type == ChatMessageType.userMessage &&
                  m.content == 'hello echo',
            );
            expect(
              hasUserMessage,
              isTrue,
              reason: 'User message "hello echo" should be in the message list',
            );

            final hasAgentMessage = lastSnapshot.any(
              (m) => m.type == ChatMessageType.agentMessage,
            );
            expect(
              hasAgentMessage,
              isTrue,
              reason: 'Agent response should appear in the message list',
            );
          } finally {
            await sub.cancel();
          }
        } finally {
          chatRepo.dispose();
        }
      } finally {
        await manager.disconnectAll();
        manager.dispose();
      }
    });
  });

  // -------------------------------------------------------------------------
  // FE2E-06: Ping through real WS
  // -------------------------------------------------------------------------

  testWidgets('FE2E-06: Ping returns pong via real WS', (tester) async {
    await tester.runAsync(() async {
      final connection = _createConnection(
        backend,
        deviceId: _isPaired
            ? _pairedDeviceId
            : 'ping-${DateTime.now().millisecondsSinceEpoch}',
      );

      try {
        await connection.connect();
        expect(connection.status, WsConnectionStatus.connected);

        final client = _createClient(connection);

        try {
          final result = await client.ping();

          expect(
            result.timestamp,
            isNotNull,
            reason: 'Ping result should include a timestamp',
          );
          // serverName and serverId may be populated depending on backend
          // implementation; at minimum the response should not throw.
        } finally {
          await client.dispose();
        }
      } finally {
        await connection.dispose();
      }
    });
  });

  // -------------------------------------------------------------------------
  // FE2E-07: Reconnect after disconnect
  // -------------------------------------------------------------------------

  testWidgets('FE2E-07: Reconnect after disconnect', (tester) async {
    expect(_isPaired, isTrue, reason: 'FE2E-01 must pass first');

    await tester.runAsync(() async {
      final server = _buildPairedServer(backend);
      final manager = WsConnectionManager();

      try {
        // 1. Connect and verify.
        await manager.connectToServer(server);

        final connBefore = manager.getConnection(_pairedServerId);
        expect(
          connBefore,
          isNotNull,
          reason: 'Connection should exist after connectToServer',
        );
        expect(
          connBefore!.status,
          WsConnectionStatus.connected,
          reason: 'Connection should be connected',
        );

        // 2. Disconnect — the manager removes the entry entirely.
        await manager.disconnectFromServer(_pairedServerId);
        expect(
          manager.getConnection(_pairedServerId),
          isNull,
          reason: 'Connection should be removed after disconnect',
        );

        // 3. Reconnect with the same server.
        await manager.connectToServer(server);

        final connAfter = manager.getConnection(_pairedServerId);
        expect(
          connAfter,
          isNotNull,
          reason: 'Connection should exist after reconnect',
        );
        expect(
          connAfter!.status,
          WsConnectionStatus.connected,
          reason: 'Connection should be connected after reconnect',
        );

        // 4. Verify the reconnected connection works by sending a ping.
        final pingResult = await connAfter.client.ping();
        expect(
          pingResult.timestamp,
          isNotNull,
          reason: 'Ping via reconnected connection should return a timestamp',
        );
      } finally {
        await manager.disconnectAll();
        manager.dispose();
      }
    });
  });

  // -------------------------------------------------------------------------
  // FE2E-08: Multi-message lifecycle
  // -------------------------------------------------------------------------

  testWidgets('FE2E-08: Multi-message lifecycle', (tester) async {
    expect(_isPaired, isTrue, reason: 'FE2E-01 must pass first');

    await tester.runAsync(() async {
      final server = _buildPairedServer(backend);
      final manager = WsConnectionManager();

      try {
        await manager.connectToServer(server);

        final conn = manager.getConnection(_pairedServerId);
        expect(conn, isNotNull, reason: 'Should be connected');
        expect(conn!.status, WsConnectionStatus.connected);

        // Create the real WsChatRepository.
        final chatRepo = WsChatRepository(
          connectionManager: manager,
          serverId: _pairedServerId,
          sessionId: backend.sessionId,
        );
        chatRepo.subscribe();

        try {
          // Listen for message snapshots.
          final messageSnapshots = <List<ChatMessage>>[];
          final sub = chatRepo.watchMessages().listen(messageSnapshots.add);

          try {
            final messages = ['first', 'second', 'third'];

            // Send 3 messages in sequence, waiting for each agent reply.
            for (var i = 0; i < messages.length; i++) {
              await chatRepo.sendMessage(backend.sessionId, messages[i]);

              final expectedAgentCount = i + 1;
              await _waitForCondition(
                () =>
                    messageSnapshots.isNotEmpty &&
                    messageSnapshots.last
                            .where(
                              (m) => m.type == ChatMessageType.agentMessage,
                            )
                            .length >=
                        expectedAgentCount,
                timeout: const Duration(seconds: 30),
              );
            }

            // Verify the final snapshot contains all user and agent messages.
            final lastSnapshot = messageSnapshots.last;

            final userMessages = lastSnapshot
                .where((m) => m.type == ChatMessageType.userMessage)
                .toList();
            final agentMessages = lastSnapshot
                .where((m) => m.type == ChatMessageType.agentMessage)
                .toList();

            expect(
              userMessages.length,
              greaterThanOrEqualTo(3),
              reason: 'Should have at least 3 user messages',
            );
            expect(
              agentMessages.length,
              greaterThanOrEqualTo(3),
              reason: 'Should have at least 3 agent responses',
            );

            // Verify user messages match the sent texts.
            for (final text in messages) {
              expect(
                userMessages.any((m) => m.content == text),
                isTrue,
                reason: 'User message "$text" should be in the message list',
              );
            }

            // Verify all agent responses have non-empty content.
            for (final agentMsg in agentMessages) {
              expect(
                agentMsg.content,
                isNotEmpty,
                reason: 'Agent response should have content',
              );
            }
          } finally {
            await sub.cancel();
          }
        } finally {
          chatRepo.dispose();
        }
      } finally {
        await manager.disconnectAll();
        manager.dispose();
      }
    });
  });
}

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/mobile_service_client.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/connection_info_badge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

// ---------------------------------------------------------------------------
// Fake MobileServiceClient to satisfy ServerConnection constructor
// ---------------------------------------------------------------------------

class _FakeMobileServiceClient implements MobileServiceClient {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

final _now = DateTime.now();

final _testServer = Server(
  id: 'srv-1',
  name: 'Test Server',
  lanAddress: '192.168.1.50',
  connectionMode: ConnectionMode.lan,
  isOnline: true,
  latencyMs: 5,
  pairedAt: _now,
);

ServerConnection _makeConnection({
  required GrpcConnectionStatus status,
  ConnectionRoute route = ConnectionRoute.lan,
  bool withEncryption = false,
}) {
  final connection = ServerConnection(
    server: _testServer,
    client: _FakeMobileServiceClient(),
    currentRoute: route,
  );
  connection.status = status;
  return connection;
}

Widget _buildBadge(ServerConnection connection) {
  return MaterialApp(
    home: Scaffold(
      body: ConnectionInfoBadge(connection: connection),
    ),
  );
}

void main() {
  testWidgets('ConnectionInfoBadge shows "LAN" for LAN route', (tester) async {
    final connection = _makeConnection(
      status: GrpcConnectionStatus.connected,
      route: ConnectionRoute.lan,
    );

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.text('LAN'), findsOneWidget);
  });

  testWidgets('ConnectionInfoBadge shows "Bridge" for bridge route',
      (tester) async {
    final connection = _makeConnection(
      status: GrpcConnectionStatus.connected,
      route: ConnectionRoute.bridge,
    );

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.text('Bridge'), findsOneWidget);
  });

  testWidgets('ConnectionInfoBadge shows LAN icon for LAN route',
      (tester) async {
    final connection = _makeConnection(
      status: GrpcConnectionStatus.connected,
      route: ConnectionRoute.lan,
    );

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.lan_outlined), findsOneWidget);
  });

  testWidgets('ConnectionInfoBadge shows cloud icon for bridge route',
      (tester) async {
    final connection = _makeConnection(
      status: GrpcConnectionStatus.connected,
      route: ConnectionRoute.bridge,
    );

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.cloud_outlined), findsOneWidget);
  });

  testWidgets('ConnectionInfoBadge hides lock icon when no encryption',
      (tester) async {
    final connection = _makeConnection(
      status: GrpcConnectionStatus.connected,
    );
    // No cipher or sharedSecret set -> hasEncryption = false.

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.lock), findsNothing);
  });

  testWidgets('ConnectionInfoBadge shows "LAN" for disconnected connection',
      (tester) async {
    final connection = _makeConnection(
      status: GrpcConnectionStatus.disconnected,
      route: ConnectionRoute.lan,
    );

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    // Route label is still based on currentRoute, even when disconnected.
    expect(find.text('LAN'), findsOneWidget);
  });
}

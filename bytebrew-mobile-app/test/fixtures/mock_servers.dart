import 'package:bytebrew_mobile/core/domain/server.dart';

final mockServers = [
  Server(
    id: 'server-1',
    name: 'MacBook Pro',
    lanAddress: '192.168.1.50',
    connectionMode: ConnectionMode.lan,
    isOnline: true,
    latencyMs: 5,
    pairedAt: DateTime.now().subtract(const Duration(days: 30)),
  ),
  Server(
    id: 'server-2',
    name: 'Desktop PC',
    lanAddress: '192.168.1.100',
    bridgeUrl: 'bytebrew.io',
    connectionMode: ConnectionMode.bridge,
    isOnline: true,
    latencyMs: 45,
    pairedAt: DateTime.now().subtract(const Duration(days: 7)),
  ),
];

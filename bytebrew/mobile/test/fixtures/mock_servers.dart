import 'package:bytebrew_mobile/core/domain/server.dart';

final mockServers = [
  Server(
    id: 'server-1',
    name: 'MacBook Pro',
    bridgeUrl: 'wss://bridge.bytebrew.ai',
    isOnline: true,
    latencyMs: 5,
    pairedAt: DateTime.now().subtract(const Duration(days: 30)),
  ),
  Server(
    id: 'server-2',
    name: 'Desktop PC',
    bridgeUrl: 'wss://bridge.bytebrew.ai',
    isOnline: true,
    latencyMs: 45,
    pairedAt: DateTime.now().subtract(const Duration(days: 7)),
  ),
];

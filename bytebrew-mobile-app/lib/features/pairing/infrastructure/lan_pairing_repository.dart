import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/pairing/domain/pairing_repository.dart';

/// [PairingRepository] that pairs with a CLI running on the local network.
///
/// Connects via WebSocket and reads the `init` message to verify reachability
/// and extract project metadata.
class LanPairingRepository implements PairingRepository {
  @override
  Future<Server> pair({
    required String serverAddress,
    required String pairingCode,
  }) async {
    // Parse address: "host" or "host:port" (default 8765).
    final parts = serverAddress.split(':');
    final host = parts[0];
    final port = parts.length > 1 ? int.tryParse(parts[1]) ?? 8765 : 8765;

    final wsUrl = 'ws://$host:$port';

    // Try connecting to verify reachability.
    final channel = WebSocketChannel.connect(Uri.parse(wsUrl));

    Map<String, dynamic> meta = {};
    try {
      await channel.ready;

      // Wait for the init message (with timeout).
      final initMsg = await channel.stream.first.timeout(
        const Duration(seconds: 5),
        onTimeout: () => throw Exception('Timeout waiting for init message'),
      );

      final json = jsonDecode(initMsg as String) as Map<String, dynamic>;
      if (json['type'] == 'init') {
        meta = json['meta'] as Map<String, dynamic>? ?? {};
      }
    } finally {
      await channel.sink.close();
    }

    return Server(
      id: 'ws-$host-$port',
      name: meta['projectName'] as String? ?? host,
      lanAddress: host,
      wsPort: port,
      isOnline: true,
      pairedAt: DateTime.now(),
    );
  }
}

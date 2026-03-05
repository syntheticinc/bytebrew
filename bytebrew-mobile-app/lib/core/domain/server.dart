/// A paired CLI server that runs the ByteBrew agent.
class Server {
  const Server({
    required this.id,
    required this.name,
    required this.lanAddress,
    this.wsPort = 8765,
    required this.isOnline,
    required this.pairedAt,
  });

  final String id;
  final String name;
  final String lanAddress;
  final int wsPort;
  final bool isOnline;
  final DateTime pairedAt;

  /// WebSocket URL for connecting to this server's CLI.
  String get wsUrl => 'ws://$lanAddress:$wsPort';

  Server copyWith({
    String? id,
    String? name,
    String? lanAddress,
    int? wsPort,
    bool? isOnline,
    DateTime? pairedAt,
  }) {
    return Server(
      id: id ?? this.id,
      name: name ?? this.name,
      lanAddress: lanAddress ?? this.lanAddress,
      wsPort: wsPort ?? this.wsPort,
      isOnline: isOnline ?? this.isOnline,
      pairedAt: pairedAt ?? this.pairedAt,
    );
  }
}

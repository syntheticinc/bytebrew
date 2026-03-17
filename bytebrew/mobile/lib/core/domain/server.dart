import 'dart:typed_data';

/// A paired CLI server that runs the ByteBrew agent.
///
/// Mobile communicates with this server through a Bridge relay via WebSocket.
class Server {
  const Server({
    required this.id,
    required this.name,
    required this.bridgeUrl,
    required this.isOnline,
    this.latencyMs = 0,
    required this.pairedAt,
    this.deviceToken,
    this.deviceId,
    this.sharedSecret,
    this.publicKey,
    this.serverPublicKey,
  });

  final String id;
  final String name;
  final String bridgeUrl;
  final bool isOnline;
  final int latencyMs;
  final DateTime pairedAt;
  final String? deviceToken;
  final String? deviceId;
  final Uint8List? sharedSecret;
  final Uint8List? publicKey;
  final Uint8List? serverPublicKey;

  /// Whether end-to-end encryption is available for this server.
  bool get hasEncryption => sharedSecret != null && sharedSecret!.isNotEmpty;

  Server copyWith({
    String? id,
    String? name,
    String? bridgeUrl,
    bool? isOnline,
    int? latencyMs,
    DateTime? pairedAt,
    String? deviceToken,
    String? deviceId,
    Uint8List? sharedSecret,
    Uint8List? publicKey,
    Uint8List? serverPublicKey,
  }) {
    return Server(
      id: id ?? this.id,
      name: name ?? this.name,
      bridgeUrl: bridgeUrl ?? this.bridgeUrl,
      isOnline: isOnline ?? this.isOnline,
      latencyMs: latencyMs ?? this.latencyMs,
      pairedAt: pairedAt ?? this.pairedAt,
      deviceToken: deviceToken ?? this.deviceToken,
      deviceId: deviceId ?? this.deviceId,
      sharedSecret: sharedSecret ?? this.sharedSecret,
      publicKey: publicKey ?? this.publicKey,
      serverPublicKey: serverPublicKey ?? this.serverPublicKey,
    );
  }
}

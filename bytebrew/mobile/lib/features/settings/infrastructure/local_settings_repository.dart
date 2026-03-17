import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/secure_key_storage.dart';
import 'package:bytebrew_mobile/features/settings/domain/settings_repository.dart';

/// Persistent implementation of [SettingsRepository].
///
/// Stores server data in [SharedPreferences].
class LocalSettingsRepository implements SettingsRepository {
  LocalSettingsRepository(this._prefs, [SecureKeyStorage? secureKeyStorage])
    : _secureKeyStorage = secureKeyStorage ?? SecureKeyStorage();

  static const _serversKey = 'saved_servers';

  final SharedPreferences _prefs;
  final SecureKeyStorage _secureKeyStorage;

  @override
  List<Server> getServers() {
    final json = _prefs.getString(_serversKey);
    if (json == null) return [];

    final list = jsonDecode(json) as List<dynamic>;
    return list.map((e) => _serverFromJson(e as Map<String, dynamic>)).toList();
  }

  @override
  Future<void> removeServer(String id) async {
    final servers = getServers().where((s) => s.id != id).toList();
    await _saveServers(servers);
  }

  /// Returns servers with their encryption keys merged from secure storage.
  @override
  Future<List<Server>> getServersWithKeys() async {
    final servers = getServers();
    final result = <Server>[];

    for (final server in servers) {
      final keys = await _secureKeyStorage.getServerKeys(server.id);
      final deviceToken = await _secureKeyStorage.getDeviceToken(server.id);

      result.add(
        server.copyWith(
          sharedSecret: keys.sharedSecret ?? server.sharedSecret,
          publicKey: keys.publicKey ?? server.publicKey,
          serverPublicKey: keys.serverPublicKey ?? server.serverPublicKey,
          deviceToken: deviceToken ?? server.deviceToken,
        ),
      );
    }

    return result;
  }

  /// Adds or replaces the [server] in local storage.
  @override
  Future<void> addServer(Server server) async {
    final servers = getServers();
    final index = servers.indexWhere((s) => s.id == server.id);
    if (index != -1) {
      servers[index] = server;
    } else {
      servers.add(server);
    }
    await _saveServers(servers);
  }

  Future<void> _saveServers(List<Server> servers) async {
    final json = jsonEncode(servers.map(_serverToJson).toList());
    await _prefs.setString(_serversKey, json);
  }

  static Map<String, dynamic> _serverToJson(Server s) {
    final map = <String, dynamic>{
      'id': s.id,
      'name': s.name,
      'bridgeUrl': s.bridgeUrl,
      'isOnline': s.isOnline,
      'latencyMs': s.latencyMs,
      'pairedAt': s.pairedAt.toIso8601String(),
    };

    if (s.deviceToken != null) map['deviceToken'] = s.deviceToken;
    if (s.deviceId != null) map['deviceId'] = s.deviceId;
    // Note: sharedSecret, publicKey, serverPublicKey are stored in
    // SecureKeyStorage, NOT in SharedPreferences.

    return map;
  }

  static Server _serverFromJson(Map<String, dynamic> json) {
    return Server(
      id: json['id'] as String,
      name: json['name'] as String,
      bridgeUrl: json['bridgeUrl'] as String? ?? '',
      isOnline: json['isOnline'] as bool? ?? false,
      latencyMs: json['latencyMs'] as int? ?? 0,
      pairedAt: DateTime.parse(json['pairedAt'] as String),
      deviceToken: json['deviceToken'] as String?,
      deviceId: json['deviceId'] as String?,
    );
  }
}

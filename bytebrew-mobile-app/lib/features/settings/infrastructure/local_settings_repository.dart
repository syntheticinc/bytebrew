import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/features/settings/domain/settings_repository.dart';

/// Persistent implementation of [SettingsRepository].
///
/// Stores server data in [SharedPreferences].
class LocalSettingsRepository implements SettingsRepository {
  LocalSettingsRepository(this._prefs);

  static const _serversKey = 'saved_servers';

  final SharedPreferences _prefs;

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

  /// Adds or replaces the [server] in local storage.
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

  static Map<String, dynamic> _serverToJson(Server s) => {
    'id': s.id,
    'name': s.name,
    'lanAddress': s.lanAddress,
    'wsPort': s.wsPort,
    'isOnline': s.isOnline,
    'pairedAt': s.pairedAt.toIso8601String(),
  };

  static Server _serverFromJson(Map<String, dynamic> json) => Server(
    id: json['id'] as String,
    name: json['name'] as String,
    lanAddress: json['lanAddress'] as String,
    wsPort: json['wsPort'] as int? ?? 8765,
    isOnline: json['isOnline'] as bool? ?? false,
    pairedAt: DateTime.parse(json['pairedAt'] as String),
  );
}

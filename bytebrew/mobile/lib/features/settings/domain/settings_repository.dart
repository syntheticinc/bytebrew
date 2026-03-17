import 'package:bytebrew_mobile/core/domain/server.dart';

/// Repository for app settings and server management.
abstract class SettingsRepository {
  /// Returns all paired servers.
  List<Server> getServers();

  /// Returns all paired servers with encryption keys from secure storage.
  Future<List<Server>> getServersWithKeys();

  /// Adds or replaces a paired [server].
  Future<void> addServer(Server server);

  /// Removes the server with the given [id].
  Future<void> removeServer(String id);
}

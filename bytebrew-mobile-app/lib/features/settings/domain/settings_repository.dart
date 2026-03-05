import 'package:bytebrew_mobile/core/domain/server.dart';

/// Repository for app settings and server management.
abstract class SettingsRepository {
  /// Returns all paired servers.
  List<Server> getServers();

  /// Removes the server with the given [id].
  Future<void> removeServer(String id);
}

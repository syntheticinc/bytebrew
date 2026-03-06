import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';

/// Provides the singleton [WsConnectionManager] for the app.
final connectionManagerProvider = Provider<WsConnectionManager>((ref) {
  final manager = WsConnectionManager();
  ref.onDispose(manager.dispose);
  return manager;
});

import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';

/// Provides the singleton [WsConnectionManager] for the app.
///
/// Bridges [ChangeNotifier] → Riverpod reactivity: when [WsConnectionManager]
/// fires [notifyListeners], all `ref.watch(connectionManagerProvider)` callers
/// rebuild. Riverpod 3 removed ChangeNotifierProvider, so we use a plain
/// [Provider] with a manual listener that calls [Ref.notifyListeners].
final connectionManagerProvider = Provider<WsConnectionManager>((ref) {
  final manager = WsConnectionManager();

  // Forward ChangeNotifier updates to Riverpod watchers.
  void onChanged() {
    // ignore: invalid_use_of_protected_member
    ref.notifyListeners();
  }

  manager.addListener(onChanged);
  ref.onDispose(() {
    manager.removeListener(onChanged);
    manager.dispose();
  });

  return manager;
});

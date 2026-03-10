import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:marionette_flutter/marionette_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app.dart';
import 'core/providers/shared_preferences_provider.dart';
import 'features/auth/application/auth_provider.dart';

void main() async {
  if (kDebugMode) {
    MarionetteBinding.ensureInitialized();
  } else {
    WidgetsFlutterBinding.ensureInitialized();
  }
  final prefs = await SharedPreferences.getInstance();

  runApp(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
        // Local dev: override Cloud API URL to LAN address.
        if (kDebugMode)
          cloudApiBaseUrlProvider.overrideWithValue('http://localhost:9700'),
      ],
      child: const ByteBrewApp(),
    ),
  );
}

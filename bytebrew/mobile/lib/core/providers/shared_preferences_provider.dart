import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';

part 'shared_preferences_provider.g.dart';

/// Provides the [SharedPreferences] instance.
///
/// Must be overridden at startup in [ProviderScope.overrides] with a
/// pre-initialized instance from [SharedPreferences.getInstance].
@Riverpod(keepAlive: true)
SharedPreferences sharedPreferences(Ref ref) {
  throw UnimplementedError(
    'sharedPreferencesProvider must be overridden at startup',
  );
}

import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/domain/auth_tokens.dart';

/// Persists authentication tokens using SharedPreferences.
class TokenStorage {
  static const _accessTokenKey = 'auth_access_token';
  static const _refreshTokenKey = 'auth_refresh_token';

  /// Saves both tokens to local storage.
  Future<void> saveTokens(AuthTokens tokens) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_accessTokenKey, tokens.accessToken);
    await prefs.setString(_refreshTokenKey, tokens.refreshToken);
  }

  /// Loads tokens from local storage, or returns null if absent.
  Future<AuthTokens?> getTokens() async {
    final prefs = await SharedPreferences.getInstance();
    final accessToken = prefs.getString(_accessTokenKey);
    final refreshToken = prefs.getString(_refreshTokenKey);

    if (accessToken == null || refreshToken == null) {
      return null;
    }

    return AuthTokens(accessToken: accessToken, refreshToken: refreshToken);
  }

  /// Removes both tokens from local storage.
  Future<void> clearTokens() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_accessTokenKey);
    await prefs.remove(_refreshTokenKey);
  }

  /// Returns true if both tokens are present in local storage.
  Future<bool> hasTokens() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.containsKey(_accessTokenKey) &&
        prefs.containsKey(_refreshTokenKey);
  }
}

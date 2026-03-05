import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../core/domain/auth_tokens.dart';
import '../domain/auth_repository.dart';

/// [AuthRepository] implementation that talks to bytebrew-cloud-api REST endpoints.
class CloudAuthRepository implements AuthRepository {
  CloudAuthRepository({required this.baseUrl, http.Client? client})
    : _client = client ?? http.Client();

  final String baseUrl;
  final http.Client _client;

  @override
  Future<AuthTokens> login(String email, String password) async {
    final response = await _post('/api/v1/auth/login', {
      'email': email,
      'password': password,
    });

    return _parseTokens(response);
  }

  @override
  Future<AuthTokens> register(String email, String password) async {
    final response = await _post('/api/v1/auth/register', {
      'email': email,
      'password': password,
    });

    return _parseTokens(response);
  }

  @override
  Future<void> logout() async {
    await _post('/api/v1/auth/logout', {});
  }

  Future<http.Response> _post(String path, Map<String, dynamic> body) async {
    final uri = Uri.parse('$baseUrl$path');
    final http.Response response;
    try {
      response = await _client
          .post(
            uri,
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode(body),
          )
          .timeout(const Duration(seconds: 10));
    } on Exception {
      throw AuthException('Server unavailable. Check your connection.');
    }

    if (response.statusCode < 200 || response.statusCode >= 300) {
      final message = _extractErrorMessage(response);
      throw AuthException(message, statusCode: response.statusCode);
    }

    return response;
  }

  AuthTokens _parseTokens(http.Response response) {
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final data = json['data'] as Map<String, dynamic>? ?? json;
    final accessToken = data['access_token'] as String?;
    final refreshToken = data['refresh_token'] as String?;

    if (accessToken == null || refreshToken == null) {
      throw const AuthException('Invalid token response from server');
    }

    return AuthTokens(accessToken: accessToken, refreshToken: refreshToken);
  }

  String _extractErrorMessage(http.Response response) {
    try {
      final json = jsonDecode(response.body) as Map<String, dynamic>;
      return json['message'] as String? ??
          json['error'] as String? ??
          'Request failed with status ${response.statusCode}';
    } catch (_) {
      return 'Request failed with status ${response.statusCode}';
    }
  }
}

/// Exception thrown when an authentication request fails.
class AuthException implements Exception {
  const AuthException(this.message, {this.statusCode});

  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

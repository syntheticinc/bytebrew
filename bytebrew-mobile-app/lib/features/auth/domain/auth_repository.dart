import '../../../core/domain/auth_tokens.dart';

/// Contract for authentication operations against the cloud API.
abstract class AuthRepository {
  /// Authenticates with email/password, returns access + refresh tokens.
  Future<AuthTokens> login(String email, String password);

  /// Creates a new account, returns access + refresh tokens.
  Future<AuthTokens> register(String email, String password);

  /// Invalidates the current session on the server.
  Future<void> logout();
}

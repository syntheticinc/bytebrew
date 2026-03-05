import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../domain/auth_repository.dart';
import '../infrastructure/cloud_auth_repository.dart';
import '../infrastructure/token_storage.dart';

part 'auth_provider.g.dart';

/// Possible states of the authentication flow.
enum AuthStatus { authenticated, unauthenticated, loading, error }

/// Immutable state for the auth provider.
class AuthState {
  const AuthState({required this.status, this.error});

  const AuthState.authenticated()
    : status = AuthStatus.authenticated,
      error = null;

  const AuthState.unauthenticated()
    : status = AuthStatus.unauthenticated,
      error = null;

  const AuthState.loading() : status = AuthStatus.loading, error = null;

  const AuthState.error(String this.error) : status = AuthStatus.error;

  final AuthStatus status;
  final String? error;
}

/// Provides the [TokenStorage] singleton.
@Riverpod(keepAlive: true)
TokenStorage tokenStorage(Ref ref) => TokenStorage();

/// Base URL for the Cloud API. Defaults to production; override in
/// [ProviderScope] for local development (e.g. `http://<LAN-IP>:60402`).
@Riverpod(keepAlive: true)
String cloudApiBaseUrl(Ref ref) => 'https://api.bytebrew.ai';

/// Provides the [AuthRepository] implementation.
@Riverpod(keepAlive: true)
AuthRepository authRepository(Ref ref) =>
    CloudAuthRepository(baseUrl: ref.watch(cloudApiBaseUrlProvider));

/// Manages authentication state across the app.
@Riverpod(keepAlive: true)
class Auth extends _$Auth {
  @override
  AuthState build() {
    _checkSavedTokens();
    return const AuthState.loading();
  }

  Future<void> _checkSavedTokens() async {
    final storage = ref.read(tokenStorageProvider);
    final hasTokens = await storage.hasTokens();

    if (hasTokens) {
      state = const AuthState.authenticated();
    } else {
      state = const AuthState.unauthenticated();
    }
  }

  /// Attempts to log in with the given credentials.
  Future<void> login(String email, String password) async {
    state = const AuthState.loading();

    try {
      final repo = ref.read(authRepositoryProvider);
      final storage = ref.read(tokenStorageProvider);
      final tokens = await repo.login(email, password);
      await storage.saveTokens(tokens);
      state = const AuthState.authenticated();
    } catch (e) {
      state = AuthState.error(e.toString());
    }
  }

  /// Attempts to register a new account.
  Future<void> register(String email, String password) async {
    state = const AuthState.loading();

    try {
      final repo = ref.read(authRepositoryProvider);
      final storage = ref.read(tokenStorageProvider);
      final tokens = await repo.register(email, password);
      await storage.saveTokens(tokens);
      state = const AuthState.authenticated();
    } catch (e) {
      state = AuthState.error(e.toString());
    }
  }

  /// Signs out and clears stored tokens.
  Future<void> logout() async {
    final repo = ref.read(authRepositoryProvider);
    final storage = ref.read(tokenStorageProvider);

    try {
      await repo.logout();
    } catch (_) {
      // Ignore server errors on logout — clear tokens regardless.
    }

    await storage.clearTokens();
    state = const AuthState.unauthenticated();
  }
}

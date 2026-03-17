import 'dart:convert';

import 'package:bytebrew_mobile/features/auth/infrastructure/cloud_auth_repository.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  const baseUrl = 'https://api.example.com';

  CloudAuthRepository createRepo(MockClient client) {
    return CloudAuthRepository(baseUrl: baseUrl, client: client);
  }

  group('login', () {
    test('returns AuthTokens on 200', () async {
      final client = MockClient((request) async {
        expect(request.url.toString(), '$baseUrl/api/v1/auth/login');
        expect(request.method, 'POST');

        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body['email'], 'user@test.com');
        expect(body['password'], 'secret123');

        return http.Response(
          jsonEncode({
            'access_token': 'access-abc',
            'refresh_token': 'refresh-xyz',
          }),
          200,
        );
      });

      final repo = createRepo(client);
      final tokens = await repo.login('user@test.com', 'secret123');

      expect(tokens.accessToken, 'access-abc');
      expect(tokens.refreshToken, 'refresh-xyz');
    });

    test('throws AuthException on 401', () async {
      final client = MockClient((_) async {
        return http.Response(
          jsonEncode({'message': 'Invalid credentials'}),
          401,
        );
      });

      final repo = createRepo(client);

      expect(
        () => repo.login('user@test.com', 'wrong'),
        throwsA(
          isA<AuthException>()
              .having((e) => e.message, 'message', 'Invalid credentials')
              .having((e) => e.statusCode, 'statusCode', 401),
        ),
      );
    });

    test(
      'throws AuthException with fallback message on non-JSON error',
      () async {
        final client = MockClient((_) async {
          return http.Response('Internal Server Error', 500);
        });

        final repo = createRepo(client);

        expect(
          () => repo.login('user@test.com', 'pass'),
          throwsA(
            isA<AuthException>().having(
              (e) => e.message,
              'message',
              'Request failed with status 500',
            ),
          ),
        );
      },
    );
  });

  group('register', () {
    test('returns AuthTokens on 200', () async {
      final client = MockClient((request) async {
        expect(request.url.toString(), '$baseUrl/api/v1/auth/register');
        expect(request.method, 'POST');

        return http.Response(
          jsonEncode({
            'access_token': 'new-access',
            'refresh_token': 'new-refresh',
          }),
          200,
        );
      });

      final repo = createRepo(client);
      final tokens = await repo.register('new@test.com', 'password1');

      expect(tokens.accessToken, 'new-access');
      expect(tokens.refreshToken, 'new-refresh');
    });

    test('throws AuthException on 409 (already exists)', () async {
      final client = MockClient((_) async {
        return http.Response(
          jsonEncode({'message': 'User already exists'}),
          409,
        );
      });

      final repo = createRepo(client);

      expect(
        () => repo.register('existing@test.com', 'pass'),
        throwsA(
          isA<AuthException>()
              .having((e) => e.message, 'message', 'User already exists')
              .having((e) => e.statusCode, 'statusCode', 409),
        ),
      );
    });
  });

  group('logout', () {
    test('completes without error on 200', () async {
      final client = MockClient((request) async {
        expect(request.url.toString(), '$baseUrl/api/v1/auth/logout');
        return http.Response('', 200);
      });

      final repo = createRepo(client);
      await expectLater(repo.logout(), completes);
    });
  });
}

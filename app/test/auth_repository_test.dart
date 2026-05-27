import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import 'package:eve_o_provit/auth/auth_repository.dart';
import 'package:eve_o_provit/auth/token_store.dart';

// The default Env.apiBaseUrl — matches the String.fromEnvironment default.
const _baseUrl = 'http://10.0.2.2:9001';

/// Fake authenticate function that echoes back the state from the authorize
/// URL, simulating a successful browser redirect with matching state.
Future<String> Function(String url, String scheme) fakeAuthenticate({
  String code = 'TEST_CODE',
  String? overrideReturnedState,
}) {
  return (String url, String scheme) async {
    final uri = Uri.parse(url);
    final sentState =
        overrideReturnedState ?? uri.queryParameters['state']!;
    return '$scheme://callback?code=$code&state=$sentState';
  };
}

/// Fake authenticate that always returns the wrong state (CSRF simulation).
Future<String> fakeAuthenticateBadState(String url, String scheme) async =>
    '$scheme://callback?code=EVIL&state=wrong-state-value';

void main() {
  late Dio dio;
  late DioAdapter adapter;
  late TokenStore store;

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
    store = TokenStore();
    // Use bare Dio (no baseUrl) + UrlRequestMatcher so full-URL calls from
    // AuthRepository match the stubs by URL path only (ignores headers/body).
    dio = Dio();
    adapter = DioAdapter(dio: dio, matcher: const UrlRequestMatcher());
  });

  // ── login() ───────────────────────────────────────────────────────────────

  group('AuthRepository.login()', () {
    test('successful: returns Character + saves tokens', () async {
      adapter.onPost(
        '$_baseUrl/auth/mobile/callback',
        (server) => server.reply(200, {
          'access_token': 'acc_abc',
          'refresh_token': 'ref_xyz',
          'expires_in': 1200,
          'character': {
            'character_id': 12345678,
            'character_name': 'Test Pilot',
            'portrait_url':
                'https://images.evetech.net/characters/12345678/portrait',
            'scopes': ['esi-location.read_location.v1'],
          },
        }),
      );

      final repo = AuthRepository(
        dio: dio,
        tokenStore: store,
        authenticate: fakeAuthenticate(code: 'TEST_CODE'),
      );

      final character = await repo.login();

      expect(character.id, 12345678);
      expect(character.name, 'Test Pilot');
      expect(character.scopes, contains('esi-location.read_location.v1'));

      expect(await store.readAccess(), 'acc_abc');
      expect(await store.readRefresh(), 'ref_xyz');
    });

    test('state mismatch: throws AuthException, no tokens saved', () async {
      // No stub needed — the repo throws before the HTTP call.
      final repo = AuthRepository(
        dio: dio,
        tokenStore: store,
        authenticate: fakeAuthenticateBadState,
      );

      await expectLater(
        () => repo.login(),
        throwsA(
          isA<AuthException>().having(
            (e) => e.message,
            'message',
            contains('state mismatch'),
          ),
        ),
      );

      expect(await store.readAccess(), isNull);
      expect(await store.readRefresh(), isNull);
    });
  });

  // ── refresh() ─────────────────────────────────────────────────────────────

  group('AuthRepository.refresh()', () {
    test('success: saves new tokens', () async {
      await store.save(access: 'old_acc', refresh: 'old_ref');

      adapter.onPost(
        '$_baseUrl/auth/mobile/refresh',
        (server) => server.reply(200, {
          'access_token': 'new_acc',
          'refresh_token': 'new_ref',
          'expires_in': 1200,
        }),
      );

      final repo = AuthRepository(
        dio: dio,
        tokenStore: store,
        authenticate: fakeAuthenticate(),
      );

      await repo.refresh();

      expect(await store.readAccess(), 'new_acc');
      expect(await store.readRefresh(), 'new_ref');
    });

    test('failure (401): clears tokens + throws AuthException', () async {
      await store.save(access: 'old_acc', refresh: 'bad_ref');

      adapter.onPost(
        '$_baseUrl/auth/mobile/refresh',
        (server) => server.throws(
          401,
          DioException(
            requestOptions:
                RequestOptions(path: '$_baseUrl/auth/mobile/refresh'),
            response: Response(
              requestOptions:
                  RequestOptions(path: '$_baseUrl/auth/mobile/refresh'),
              statusCode: 401,
            ),
            type: DioExceptionType.badResponse,
          ),
        ),
      );

      final repo = AuthRepository(
        dio: dio,
        tokenStore: store,
        authenticate: fakeAuthenticate(),
      );

      await expectLater(
        () => repo.refresh(),
        throwsA(isA<AuthException>()),
      );

      expect(await store.readAccess(), isNull);
      expect(await store.readRefresh(), isNull);
    });

    test('no refresh token: throws without any network call', () async {
      final repo = AuthRepository(
        dio: dio,
        tokenStore: store,
        authenticate: fakeAuthenticate(),
      );

      await expectLater(
        () => repo.refresh(),
        throwsA(isA<AuthException>()),
      );
    });
  });

  // ── logout() ──────────────────────────────────────────────────────────────

  group('AuthRepository.logout()', () {
    test('clears stored tokens', () async {
      await store.save(access: 'acc', refresh: 'ref');

      final repo = AuthRepository(
        dio: dio,
        tokenStore: store,
        authenticate: fakeAuthenticate(),
      );

      await repo.logout();

      expect(await store.readAccess(), isNull);
      expect(await store.readRefresh(), isNull);
    });
  });
}

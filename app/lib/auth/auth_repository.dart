import 'dart:math';

import 'package:dio/dio.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';

import '../core/env.dart';
import 'character.dart';
import 'pkce.dart';
import 'token_store.dart';

/// Thrown when an authentication operation fails.
class AuthException implements Exception {
  const AuthException(this.message);

  final String message;

  @override
  String toString() => 'AuthException: $message';
}

/// Abstraction over the platform browser-auth call.
///
/// The default implementation delegates to [FlutterWebAuth2.authenticate].
/// In tests, inject a fake that returns a canned redirect URL so no platform
/// is needed.
typedef AuthenticateFn = Future<String> Function(String url, String scheme);

/// Handles EVE SSO OAuth2 PKCE login, token refresh, and logout.
///
/// [dio] is used **only** for the auth endpoints (mobile/callback, mobile/refresh).
/// This is a bare Dio instance **without** the Bearer interceptor, which
/// prevents circular refresh loops.  The app-level interceptor-equipped Dio
/// lives in [DioClient] and is wired separately.
class AuthRepository {
  AuthRepository({
    required Dio dio,
    required TokenStore tokenStore,
    AuthenticateFn? authenticate,
  })  : _dio = dio,
        _store = tokenStore,
        _authenticate = authenticate ??
            ((url, scheme) => FlutterWebAuth2.authenticate(
                  url: url,
                  callbackUrlScheme: scheme,
                ));

  final Dio _dio;
  final TokenStore _store;
  final AuthenticateFn _authenticate;

  static const _callbackScheme = 'eveauth-eveoprovit';

  /// Returns `true` if a refresh token is present (session may be resumed).
  Future<bool> hasSession() => _store.hasSession();

  /// Full EVE SSO PKCE login flow.
  ///
  /// Opens the system browser, waits for the redirect, exchanges the code for
  /// tokens, persists them, and returns the authenticated [Character].
  Future<Character> login() async {
    final pkce = generatePkce();
    final state = _randomState();

    final authorizeUrl = Uri.https('login.eveonline.com', '/v2/oauth/authorize', {
      'response_type': 'code',
      'redirect_uri': Env.redirectUri,
      'client_id': Env.eveClientId,
      'scope': Env.scopes.join(' '),
      'state': state,
      'code_challenge': pkce.challenge,
      'code_challenge_method': 'S256',
    }).toString();

    final result = await _authenticate(authorizeUrl, _callbackScheme);
    final resultUri = Uri.parse(result);

    final returnedState = resultUri.queryParameters['state'];
    if (returnedState != state) {
      throw const AuthException('state mismatch');
    }

    final code = resultUri.queryParameters['code'];
    if (code == null || code.isEmpty) {
      throw const AuthException('missing code in redirect');
    }

    final Response response;
    try {
      response = await _dio.post(
        '${Env.apiBaseUrl}/auth/mobile/callback',
        data: {
          'code': code,
          'code_verifier': pkce.verifier,
        },
      );
    } on DioException catch (e) {
      throw AuthException(
        'callback failed: ${e.response?.statusCode ?? e.message}',
      );
    }

    if (response.statusCode != 200) {
      throw AuthException('callback returned ${response.statusCode}');
    }

    final body = response.data as Map<String, dynamic>;
    await _store.save(
      access: body['access_token'] as String,
      refresh: body['refresh_token'] as String,
    );

    return Character.fromJson(body['character'] as Map<String, dynamic>);
  }

  /// Attempts a silent token refresh using the stored refresh token.
  ///
  /// On failure, clears tokens and rethrows.
  Future<void> refresh() async {
    final refreshToken = await _store.readRefresh();
    if (refreshToken == null) {
      throw const AuthException('no refresh token available');
    }

    try {
      final response = await _dio.post(
        '${Env.apiBaseUrl}/auth/mobile/refresh',
        data: {'refresh_token': refreshToken},
      );

      if (response.statusCode != 200) {
        await _store.clear();
        throw AuthException('refresh returned ${response.statusCode}');
      }

      final body = response.data as Map<String, dynamic>;
      await _store.save(
        access: body['access_token'] as String,
        refresh: body['refresh_token'] as String,
      );
    } on DioException catch (e) {
      await _store.clear();
      throw AuthException(
        'refresh failed: ${e.response?.statusCode ?? e.message}',
      );
    }
  }

  /// Clears tokens, effectively logging out.
  Future<void> logout() => _store.clear();

  // ── Helpers ──────────────────────────────────────────────────────────────

  static String _randomState() {
    final rng = Random.secure();
    final bytes = List<int>.generate(16, (_) => rng.nextInt(256));
    // Base64url-encode then strip padding → URL-safe state value
    return _base64UrlEncode(bytes);
  }

  static String _base64UrlEncode(List<int> bytes) {
    const chars =
        'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
    final buf = StringBuffer();
    for (var i = 0; i < bytes.length; i += 3) {
      final b0 = bytes[i];
      final b1 = i + 1 < bytes.length ? bytes[i + 1] : 0;
      final b2 = i + 2 < bytes.length ? bytes[i + 2] : 0;
      buf.write(chars[(b0 >> 2) & 0x3f]);
      buf.write(chars[((b0 << 4) | (b1 >> 4)) & 0x3f]);
      if (i + 1 < bytes.length) buf.write(chars[((b1 << 2) | (b2 >> 6)) & 0x3f]);
      if (i + 2 < bytes.length) buf.write(chars[b2 & 0x3f]);
    }
    return buf.toString();
  }
}

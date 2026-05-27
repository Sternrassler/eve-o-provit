import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Thin wrapper around [FlutterSecureStorage] for persisting EVE SSO tokens.
///
/// Keys are stable constants so callers never hard-code strings.
/// All operations are async; no business logic lives here.
class TokenStore {
  TokenStore({FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  static const _keyAccess = 'eve_access_token';
  static const _keyRefresh = 'eve_refresh_token';

  /// Persists [access] and [refresh] tokens to secure storage.
  Future<void> save({required String access, required String refresh}) async {
    await _storage.write(key: _keyAccess, value: access);
    await _storage.write(key: _keyRefresh, value: refresh);
  }

  /// Returns the stored access token, or `null` if absent.
  Future<String?> readAccess() => _storage.read(key: _keyAccess);

  /// Returns the stored refresh token, or `null` if absent.
  Future<String?> readRefresh() => _storage.read(key: _keyRefresh);

  /// Deletes both tokens from secure storage.
  Future<void> clear() async {
    await _storage.delete(key: _keyAccess);
    await _storage.delete(key: _keyRefresh);
  }

  /// Returns `true` if a refresh token is present (session is active).
  Future<bool> hasSession() async {
    final refresh = await readRefresh();
    return refresh != null;
  }
}

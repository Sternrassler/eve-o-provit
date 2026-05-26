import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/env.dart';
import 'auth_providers.dart';
import 'auth_repository.dart';
import 'character.dart';

// ---------------------------------------------------------------------------
// AuthState — sealed class hierarchy
// ---------------------------------------------------------------------------

/// Base class for authentication states.
sealed class AuthState {
  const AuthState();
}

/// User is not authenticated.
class Unauthenticated extends AuthState {
  const Unauthenticated();
}

/// User is authenticated. [character] may be null after a silent restore
/// (the refresh endpoint does not return character data).
class Authenticated extends AuthState {
  const Authenticated({this.character});

  final Character? character;

  @override
  String toString() => 'Authenticated(character: $character)';
}

// ---------------------------------------------------------------------------
// AuthController — AsyncNotifier<AuthState>
// ---------------------------------------------------------------------------

/// Notifier that owns the authentication lifecycle.
///
/// On [build], it checks for an existing session via [AuthRepository.hasSession].
/// If a session is found, it attempts a silent refresh.  A successful refresh
/// yields [Authenticated] with a null character (no character data is returned
/// by the refresh endpoint — that's acceptable for v1; the app can prompt
/// re-login or fetch character data lazily).
///
/// Consumers watch [authControllerProvider] and branch on
/// [AuthState] (Unauthenticated vs Authenticated).
class AuthController extends AsyncNotifier<AuthState> {
  @override
  Future<AuthState> build() async {
    final repo = ref.read(authRepositoryProvider);

    final hasSession = await repo.hasSession();
    if (!hasSession) return const Unauthenticated();

    // Attempt silent token refresh — failure is non-fatal on startup
    try {
      await repo.refresh();
      return const Authenticated();
    } catch (_) {
      // Refresh failed (e.g. expired token) → tokens already cleared by repo
      return const Unauthenticated();
    }
  }

  /// Initiates the full EVE SSO PKCE login flow.
  Future<void> login() async {
    state = const AsyncLoading();
    final repo = ref.read(authRepositoryProvider);
    state = await AsyncValue.guard(() async {
      final character = await repo.login();
      return Authenticated(character: character);
    });
  }

  /// Logs out by clearing tokens.
  Future<void> logout() async {
    final repo = ref.read(authRepositoryProvider);
    await repo.logout();
    state = const AsyncData(Unauthenticated());
  }
}

/// Provider for [AuthController].
final authControllerProvider =
    AsyncNotifierProvider<AuthController, AuthState>(AuthController.new);

// ---------------------------------------------------------------------------
// AuthRepository provider
//
// ARCHITECTURE NOTE — Avoiding circular Dio/Auth dependency:
//   authRepositoryProvider  → uses a plain Dio (no interceptor)
//   dioProvider             → uses an interceptor-equipped Dio,
//                              calls authRepositoryProvider for refresh
//
// authRepositoryProvider does NOT read dioProvider, so no provider cycle exists.
// ---------------------------------------------------------------------------

/// Provider for [AuthRepository].
///
/// Uses a plain Dio instance without the Bearer/refresh interceptor so that
/// auth calls (mobile/callback, mobile/refresh) never trigger the interceptor.
/// Shares the [tokenStoreProvider] so both auth and app Dio use one store.
final authRepositoryProvider = Provider<AuthRepository>((ref) {
  final authDio = Dio(
    BaseOptions(
      baseUrl: Env.apiBaseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 15),
    ),
  );

  final store = ref.watch(tokenStoreProvider);

  return AuthRepository(
    dio: authDio,
    tokenStore: store,
  );
});

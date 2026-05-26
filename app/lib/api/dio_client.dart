// ARCHITECTURE — Two-Dio split to avoid circular dependency:
//
//   authRepositoryProvider  →  uses [_authDio]: a plain Dio with no interceptor.
//                               Auth calls (mobile/callback, mobile/refresh)
//                               never go through the Bearer interceptor.
//
//   dioProvider             →  uses an interceptor-equipped Dio for all
//                               app-level API calls (requires Bearer header +
//                               automatic 401 refresh+retry).  It reads
//                               [authRepositoryProvider] for the refresh call,
//                               but authRepositoryProvider does NOT read
//                               dioProvider, so the Riverpod provider graph
//                               has no cycle.

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_controller.dart';
import '../auth/auth_repository.dart';
import '../auth/token_store.dart';
import '../core/env.dart';

/// The refresh endpoint path — used to guard against infinite refresh loops.
const _refreshPath = '/auth/mobile/refresh';

/// Builds an interceptor-equipped [Dio] for the application API.
///
/// - [store] provides the current access / refresh tokens.
/// - [getAuthRepo] is a lazy factory so the [AuthRepository] is only resolved
///   when an actual 401 occurs, keeping startup fast.
Dio buildDio(TokenStore store, AuthRepository Function() getAuthRepo) {
  final dio = Dio(
    BaseOptions(
      baseUrl: Env.apiBaseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 15),
    ),
  );

  dio.interceptors.add(
    InterceptorsWrapper(
      // ── Request: attach Bearer token ────────────────────────────────────
      onRequest: (options, handler) async {
        final access = await store.readAccess();
        if (access != null) {
          options.headers['Authorization'] = 'Bearer $access';
        }
        handler.next(options);
      },

      // ── Error: on 401 attempt ONE silent refresh then retry ──────────────
      onError: (DioException error, handler) async {
        final response = error.response;

        // Only attempt refresh for HTTP 401 on non-refresh endpoints.
        // The guard prevents an infinite refresh→401→refresh loop.
        if (response?.statusCode != 401) {
          handler.next(error);
          return;
        }

        final requestPath = error.requestOptions.path;
        if (requestPath.contains(_refreshPath)) {
          // The refresh call itself returned 401 → clear tokens, propagate.
          await store.clear();
          handler.next(error);
          return;
        }

        // Check that we haven't already retried this request.
        final alreadyRetried =
            error.requestOptions.extra['_retried'] as bool? ?? false;
        if (alreadyRetried) {
          handler.next(error);
          return;
        }

        try {
          await getAuthRepo().refresh();
        } catch (_) {
          // Refresh failed — tokens already cleared by AuthRepository.refresh()
          handler.next(error);
          return;
        }

        // Retry the original request once with the new access token.
        final newAccess = await store.readAccess();
        final retryOptions = error.requestOptions
          ..headers['Authorization'] =
              newAccess != null ? 'Bearer $newAccess' : null
          ..extra['_retried'] = true;

        try {
          final retryResponse = await dio.fetch<dynamic>(retryOptions);
          handler.resolve(retryResponse);
        } on DioException catch (retryError) {
          handler.next(retryError);
        }
      },
    ),
  );

  return dio;
}

// ---------------------------------------------------------------------------
// Riverpod providers
// ---------------------------------------------------------------------------

/// Provider exposing a [TokenStore] instance.
///
/// Exposed as a provider so it can be overridden in tests.
final tokenStoreProvider = Provider<TokenStore>((_) => TokenStore());

/// Provider exposing the interceptor-equipped [Dio] for app API calls.
///
/// Reads [authRepositoryProvider] lazily (only on 401) to avoid any
/// initialization-order issues between this provider and the auth repo.
final dioProvider = Provider<Dio>((ref) {
  final store = ref.watch(tokenStoreProvider);
  return buildDio(
    store,
    // Lazy getter — resolved only when a 401 needs handling.
    () => ref.read(authRepositoryProvider),
  );
});

import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/auth/auth_repository.dart';
import 'package:eve_o_provit/auth/token_store.dart';

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

/// A minimal [HttpClientAdapter] that executes a [_handler] callback to
/// produce each response.  Allows sequential or state-dependent responses
/// without the http_mock_adapter "last-match-wins" limitation.
class _CallbackAdapter implements HttpClientAdapter {
  _CallbackAdapter(this._handler);

  final ResponseBody Function(RequestOptions options) _handler;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async =>
      _handler(options);

  @override
  void close({bool force = false}) {}
}

/// Minimal test double for [AuthRepository].  [refresh] either saves new
/// tokens or clears + throws, depending on [shouldFail].
class _FakeAuthRepo extends AuthRepository {
  _FakeAuthRepo({required this.shouldFail, required TokenStore store})
      : _store = store,
        super(
          dio: Dio(BaseOptions(baseUrl: 'http://test.local')),
          tokenStore: store,
          authenticate: (url, scheme) async => throw UnimplementedError(),
        );

  final bool shouldFail;
  final TokenStore _store;

  @override
  Future<void> refresh() async {
    if (shouldFail) {
      await _store.clear();
      throw const AuthException('refresh failed');
    }
    await _store.save(access: 'refreshed_acc', refresh: 'refreshed_ref');
  }
}

/// Builds a test [Dio] with the same request/error interceptor logic as
/// [buildDio] in `lib/api/dio_client.dart`, wired to [store] and [fakeRepo].
Dio _buildTestDio(TokenStore store, AuthRepository fakeRepo) {
  final dio = Dio(BaseOptions(baseUrl: 'http://test.local'));

  dio.interceptors.add(
    InterceptorsWrapper(
      // ── Request: attach Bearer ────────────────────────────────────────
      onRequest: (options, handler) async {
        final access = await store.readAccess();
        if (access != null) {
          options.headers['Authorization'] = 'Bearer $access';
        }
        handler.next(options);
      },

      // ── Error: on 401 attempt one refresh + retry ─────────────────────
      onError: (DioException error, handler) async {
        if (error.response?.statusCode != 401) {
          handler.next(error);
          return;
        }
        if (error.requestOptions.path.contains('/auth/mobile/refresh')) {
          await store.clear();
          handler.next(error);
          return;
        }
        final alreadyRetried =
            error.requestOptions.extra['_retried'] as bool? ?? false;
        if (alreadyRetried) {
          handler.next(error);
          return;
        }
        try {
          await fakeRepo.refresh();
        } catch (_) {
          handler.next(error);
          return;
        }
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
// Tests
// ---------------------------------------------------------------------------

void main() {
  late TokenStore store;

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
    store = TokenStore();
  });

  // ── Bearer header attachment ─────────────────────────────────────────────

  group('Request interceptor', () {
    test('attaches Bearer header when access token is present', () async {
      await store.save(access: 'my_access_token', refresh: 'ref');

      final fakeRepo = _FakeAuthRepo(shouldFail: false, store: store);
      final dio = _buildTestDio(store, fakeRepo);

      // Capture the Authorization header as it reaches the transport layer.
      String? capturedAuth;
      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedAuth = options.headers['Authorization'] as String?;
        return ResponseBody.fromString('{}', 200,
            headers: {
              Headers.contentTypeHeader: ['application/json'],
            });
      });

      await dio.get('/api/market');
      expect(capturedAuth, 'Bearer my_access_token');
    });

    test('omits Authorization header when no access token is stored', () async {
      // store is empty — token not saved
      final fakeRepo = _FakeAuthRepo(shouldFail: false, store: store);
      final dio = _buildTestDio(store, fakeRepo);

      String? capturedAuth = 'SENTINEL'; // non-null sentinel
      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedAuth = options.headers['Authorization'] as String?;
        return ResponseBody.fromString('{}', 200,
            headers: {
              Headers.contentTypeHeader: ['application/json'],
            });
      });

      await dio.get('/api/market');
      expect(capturedAuth, isNull);
    });
  });

  // ── 401 → refresh → retry ─────────────────────────────────────────────────

  group('Error interceptor (401 handling)', () {
    test('401 triggers refresh + one retry; retried request succeeds',
        () async {
      await store.save(access: 'expired_acc', refresh: 'valid_ref');

      final fakeRepo = _FakeAuthRepo(shouldFail: false, store: store);
      final dio = _buildTestDio(store, fakeRepo);

      var callCount = 0;
      dio.httpClientAdapter = _CallbackAdapter((options) {
        callCount++;
        if (callCount == 1) {
          // First request → 401
          return ResponseBody.fromString('Unauthorized', 401);
        }
        // Second request (retry) → 200
        return ResponseBody.fromString('{"result":"ok"}', 200,
            headers: {
              Headers.contentTypeHeader: ['application/json'],
            });
      });

      final response = await dio.get('/api/data');
      expect(response.statusCode, 200);
      expect(callCount, 2); // original + one retry
      expect(await store.readAccess(), 'refreshed_acc');
    });

    test('401 + refresh failure: clears tokens and propagates DioException',
        () async {
      await store.save(access: 'expired_acc', refresh: 'bad_ref');

      final fakeRepo = _FakeAuthRepo(shouldFail: true, store: store);
      final dio = _buildTestDio(store, fakeRepo);

      dio.httpClientAdapter = _CallbackAdapter((options) {
        return ResponseBody.fromString('Unauthorized', 401);
      });

      await expectLater(
        () => dio.get('/api/data'),
        throwsA(isA<DioException>()),
      );

      // Tokens must be cleared by the failed refresh.
      expect(await store.readAccess(), isNull);
      expect(await store.readRefresh(), isNull);
    });

    test('does not retry more than once (alreadyRetried guard)', () async {
      await store.save(access: 'expired_acc', refresh: 'valid_ref');

      final fakeRepo = _FakeAuthRepo(shouldFail: false, store: store);
      final dio = _buildTestDio(store, fakeRepo);

      var callCount = 0;
      dio.httpClientAdapter = _CallbackAdapter((options) {
        callCount++;
        // Always return 401 — the retry guard must prevent a loop.
        return ResponseBody.fromString('Unauthorized', 401);
      });

      await expectLater(
        () => dio.get('/api/data'),
        throwsA(isA<DioException>()),
      );

      // First call + one retry = 2 total; loop guard must stop further retries.
      expect(callCount, 2);
    });

    test(
        '401 on the refresh path itself: does NOT retry and clears tokens',
        () async {
      await store.save(access: 'expired_acc', refresh: 'stale_ref');

      // shouldFail is irrelevant here — refresh() must never be invoked.
      final fakeRepo = _FakeAuthRepo(shouldFail: false, store: store);
      final dio = _buildTestDio(store, fakeRepo);

      var callCount = 0;
      dio.httpClientAdapter = _CallbackAdapter((options) {
        callCount++;
        return ResponseBody.fromString('Unauthorized', 401);
      });

      await expectLater(
        () => dio.get('/auth/mobile/refresh'),
        throwsA(isA<DioException>()),
      );

      // No retry — the request to the refresh endpoint itself must not loop.
      expect(callCount, 1);
      // Tokens cleared because the refresh-path 401 means the session is dead.
      expect(await store.readAccess(), isNull);
      expect(await store.readRefresh(), isNull);
    });
  });
}

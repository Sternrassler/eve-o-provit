import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/auth/token_store.dart';

void main() {
  // flutter_secure_storage ^10 ships TestFlutterSecureStoragePlatform and
  // exposes FlutterSecureStorage.setMockInitialValues() which replaces the
  // platform instance with an in-memory implementation. No device needed.
  late TokenStore store;

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
    store = TokenStore();
  });

  group('TokenStore', () {
    test('readAccess() and readRefresh() return saved values after save()', () async {
      await store.save(access: 'acc_123', refresh: 'ref_456');

      expect(await store.readAccess(), 'acc_123');
      expect(await store.readRefresh(), 'ref_456');
    });

    test('hasSession() is false before save()', () async {
      expect(await store.hasSession(), isFalse);
    });

    test('hasSession() is true after save()', () async {
      await store.save(access: 'acc_abc', refresh: 'ref_xyz');
      expect(await store.hasSession(), isTrue);
    });

    test('clear() removes both tokens', () async {
      await store.save(access: 'acc_to_clear', refresh: 'ref_to_clear');
      await store.clear();

      expect(await store.readAccess(), isNull);
      expect(await store.readRefresh(), isNull);
    });

    test('hasSession() is false after clear()', () async {
      await store.save(access: 'acc', refresh: 'ref');
      await store.clear();

      expect(await store.hasSession(), isFalse);
    });
  });
}

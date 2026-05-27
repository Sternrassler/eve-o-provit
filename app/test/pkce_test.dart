import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/auth/pkce.dart';

void main() {
  group('generatePkce()', () {
    test('returns a verifier of valid length', () {
      final p = generatePkce();
      expect(p.verifier.length, greaterThanOrEqualTo(43));
      expect(p.verifier.length, lessThanOrEqualTo(128));
    });

    test('verifier contains only URL-safe unreserved chars', () {
      final p = generatePkce();
      expect(p.verifier, matches(RegExp(r'^[A-Za-z0-9\-._~]+$')));
    });

    test('challenge equals base64url(sha256(verifier)) without padding', () {
      final p = generatePkce();
      final digest = sha256.convert(utf8.encode(p.verifier));
      final expected = base64Url.encode(digest.bytes).replaceAll('=', '');
      expect(p.challenge, equals(expected));
    });

    test('two calls produce different verifiers (randomness)', () {
      final p1 = generatePkce();
      final p2 = generatePkce();
      expect(p1.verifier, isNot(equals(p2.verifier)));
    });
  });
}

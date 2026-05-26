import 'dart:convert';
import 'dart:math';

import 'package:crypto/crypto.dart';

/// Result type for PKCE code verifier + challenge pair.
typedef Pkce = ({String verifier, String challenge});

/// Generates a PKCE [code_verifier] and the matching [code_challenge] (S256).
///
/// The verifier is built by base64url-encoding 32 cryptographically random
/// bytes (yielding 44 chars) then stripping any `=` padding, resulting in 43
/// URL-safe unreserved characters (`[A-Za-z0-9-_]` — a strict subset of the
/// full unreserved set `[A-Za-z0-9-._~]` required by RFC 7636 / EVE SSO).
///
/// The challenge is `base64url( SHA-256(ASCII(verifier)) )` with padding
/// stripped, as required by RFC 7636 §4.2.
Pkce generatePkce() {
  final rng = Random.secure();
  // 32 random bytes → 44 base64url chars → strip '=' → 43 chars verifier
  final bytes = List<int>.generate(32, (_) => rng.nextInt(256));
  final verifier = base64Url.encode(bytes).replaceAll('=', '');

  final digest = sha256.convert(utf8.encode(verifier));
  final challenge = base64Url.encode(digest.bytes).replaceAll('=', '');

  return (verifier: verifier, challenge: challenge);
}

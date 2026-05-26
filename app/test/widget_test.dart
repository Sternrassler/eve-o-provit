// Smoke test: verifies that EveOProvitApp renders the LoginScreen when
// the auth state is overridden to Unauthenticated.
//
// The test does NOT require a real backend or token store — it overrides
// authControllerProvider (and tokenStoreProvider) with in-memory stubs.

import 'package:eve_o_provit/auth/auth_controller.dart';
import 'package:eve_o_provit/auth/auth_providers.dart';
import 'package:eve_o_provit/auth/token_store.dart';
import 'package:eve_o_provit/main.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows LoginScreen when unauthenticated', (tester) async {
    // Override authControllerProvider so it immediately returns Unauthenticated,
    // and tokenStoreProvider so no secure storage is touched.
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          // Prevent the real TokenStore from hitting flutter_secure_storage.
          tokenStoreProvider.overrideWithValue(TokenStore()),

          // Force the auth state to Unauthenticated — no network calls.
          authControllerProvider.overrideWith(() => _FakeAuthController()),
        ],
        child: const EveOProvitApp(),
      ),
    );

    // Let the router evaluate the redirect (one frame for async build).
    await tester.pumpAndSettle();

    // The LoginScreen must be visible.
    expect(find.text('Login mit EVE'), findsOneWidget);
  });
}

// ---------------------------------------------------------------------------
// Stub AuthController — always returns Unauthenticated synchronously.
// ---------------------------------------------------------------------------

class _FakeAuthController extends AuthController {
  @override
  Future<AuthState> build() async => const Unauthenticated();
}

/// App router — go_router configuration with auth-aware redirect and nav shell.
///
/// Routes:
///   /login    — LoginScreen  (outside shell, no navigation bar)
///   /trading  — TradingScreen (shell destination 0)
///   /character — CharacterScreen (shell destination 1)
///
/// Redirect logic:
///   • Unauthenticated → /login
///   • Authenticated + on /login → /trading
///
/// The router re-evaluates redirects whenever [authControllerProvider] emits a
/// new state via a [Listenable] bridge (ProviderListenable adapter pattern).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../auth/auth_controller.dart';
import '../auth/login_screen.dart';
import '../features/character/character_screen.dart';
import '../features/trading/trading_screen.dart';

// ---------------------------------------------------------------------------
// Auth-state Listenable bridge
// ---------------------------------------------------------------------------
//
// go_router's [refreshListenable] requires a [Listenable].
// Riverpod providers are not natively Listenable, so we bridge them via a
// [ChangeNotifier] that notifies whenever the auth state changes.

/// A [ChangeNotifier] that re-notifies go_router whenever the Riverpod
/// [authControllerProvider] state changes.
class _AuthStateListenable extends ChangeNotifier {
  _AuthStateListenable(Ref ref) : _ref = ref {
    // Listen immediately — keeps notifying as long as this object lives.
    _ref.listen<AsyncValue<AuthState>>(
      authControllerProvider,
      (prev, next) => notifyListeners(),
    );
  }

  final Ref _ref;
}

// ---------------------------------------------------------------------------
// goRouterProvider
// ---------------------------------------------------------------------------

/// Provides the configured [GoRouter] instance.
///
/// Depends on [authControllerProvider] (via the Listenable bridge) so the
/// router automatically re-runs the redirect on every auth state change.
final goRouterProvider = Provider<GoRouter>((ref) {
  final listenable = _AuthStateListenable(ref);

  return GoRouter(
    initialLocation: '/trading',
    refreshListenable: listenable,

    // ── Auth redirect ───────────────────────────────────────────────────────
    redirect: (context, state) {
      final authAsync = ref.read(authControllerProvider);

      // While loading, don't redirect — let the current route stand.
      if (authAsync.isLoading) return null;

      final isAuthenticated = authAsync.value is Authenticated;
      final onLogin = state.matchedLocation == '/login';

      if (!isAuthenticated && !onLogin) return '/login';
      if (isAuthenticated && onLogin) return '/trading';
      return null; // no redirect needed
    },

    routes: [
      // ── Login ─────────────────────────────────────────────────────────────
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),

      // ── Shell: Trading + Character ─────────────────────────────────────────
      ShellRoute(
        builder: (context, state, child) =>
            _AppShell(location: state.matchedLocation, child: child),
        routes: [
          GoRoute(
            path: '/trading',
            builder: (context, state) => const TradingScreen(),
          ),
          GoRoute(
            path: '/character',
            builder: (context, state) => const CharacterScreen(),
          ),
        ],
      ),
    ],
  );
});

// ---------------------------------------------------------------------------
// Navigation shell — bottom NavigationBar
// ---------------------------------------------------------------------------

class _AppShell extends StatelessWidget {
  const _AppShell({required this.location, required this.child});

  final String location;
  final Widget child;

  static const _destinations = [
    NavigationDestination(
      icon: Icon(Icons.currency_exchange_rounded),
      label: 'Trading',
    ),
    NavigationDestination(
      icon: Icon(Icons.person_outline_rounded),
      selectedIcon: Icon(Icons.person_rounded),
      label: 'Character',
    ),
  ];

  int get _selectedIndex {
    if (location.startsWith('/character')) return 1;
    return 0; // /trading is default
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: (index) {
          switch (index) {
            case 0:
              context.go('/trading');
            case 1:
              context.go('/character');
          }
        },
        destinations: _destinations,
      ),
    );
  }
}

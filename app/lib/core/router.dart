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
import '../features/mining/mining_screen.dart';
import '../features/trading/hauling_screen.dart';
import '../features/trading/hub_comparison_screen.dart';
import '../features/trading/roi_calculator_screen.dart';
import '../features/trading/sell_assets_screen.dart';
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
            path: '/hub-comparison',
            builder: (context, state) => const HubComparisonScreen(),
          ),
          GoRoute(
            path: '/roi-calculator',
            builder: (context, state) => const RoiCalculatorScreen(),
          ),
          GoRoute(
            path: '/hauling',
            builder: (context, state) => const HaulingScreen(),
          ),
          GoRoute(
            path: '/sell-assets',
            builder: (context, state) => const SellAssetsScreen(),
          ),
          GoRoute(
            path: '/mining',
            builder: (context, state) => const MiningScreen(),
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

class _AppShell extends ConsumerWidget {
  const _AppShell({required this.location, required this.child});

  final String location;
  final Widget child;

  // Trading + Character are real routes; "Abmelden" is an action (it opens a
  // confirm dialog instead of navigating), kept here so logout lives next to
  // the primary navigation rather than hidden in a screen's AppBar.
  static const _destinations = [
    NavigationDestination(
      icon: Icon(Icons.currency_exchange_rounded),
      label: 'Trading',
    ),
    NavigationDestination(
      icon: Icon(Icons.compare_arrows_rounded),
      label: 'Hub-Vergleich',
    ),
    NavigationDestination(
      icon: Icon(Icons.auto_graph_rounded),
      label: 'ROI',
    ),
    NavigationDestination(
      icon: Icon(Icons.local_shipping_rounded),
      label: 'Hauling',
    ),
    NavigationDestination(
      icon: Icon(Icons.sell_rounded),
      label: 'Verkaufen',
    ),
    NavigationDestination(
      icon: Icon(Icons.diamond_rounded),
      label: 'Mining',
    ),
    NavigationDestination(
      icon: Icon(Icons.person_outline_rounded),
      selectedIcon: Icon(Icons.person_rounded),
      label: 'Character',
    ),
    NavigationDestination(
      icon: Icon(Icons.logout_rounded),
      label: 'Abmelden',
    ),
  ];

  int get _selectedIndex {
    if (location.startsWith('/hub-comparison')) return 1;
    if (location.startsWith('/roi-calculator')) return 2;
    if (location.startsWith('/hauling')) return 3;
    if (location.startsWith('/sell-assets')) return 4;
    if (location.startsWith('/mining')) return 5;
    if (location.startsWith('/character')) return 6;
    return 0; // /trading is default; "Abmelden" is never a selected state
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: (index) {
          switch (index) {
            case 0:
              context.go('/trading');
            case 1:
              context.go('/hub-comparison');
            case 2:
              context.go('/roi-calculator');
            case 3:
              context.go('/hauling');
            case 4:
              context.go('/sell-assets');
            case 5:
              context.go('/mining');
            case 6:
              context.go('/character');
            case 7:
              _confirmLogout(context, ref);
          }
        },
        destinations: _destinations,
      ),
    );
  }

  Future<void> _confirmLogout(BuildContext context, WidgetRef ref) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Abmelden?'),
        content: const Text(
          'Du wirst von EVE-O Provit abgemeldet und musst dich erneut anmelden.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Abbrechen'),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Abmelden'),
          ),
        ],
      ),
    );
    if (confirmed ?? false) {
      await ref.read(authControllerProvider.notifier).logout();
    }
  }
}

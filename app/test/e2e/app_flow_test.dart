/// Automated end-to-end / integration-style widget tests for the full app flow.
///
/// ## Placement rationale
/// These tests live under `test/` (not `integration_test/`) and run with
/// plain `flutter test` — no physical device or emulator required.
/// The `integration_test/` package requires a device runner; since we mock
/// ALL backend/auth/storage layers via Riverpod provider overrides and invoke
/// no platform plugins, `flutter test` headlessly covers all scenarios.
///
/// ## What is tested
/// 1. Auth gate — Unauthenticated → LoginScreen; Authenticated → TradingScreen
/// 2. Adaptive layout landscape (≥840 dp) → two-pane (RouteList + detail pane)
/// 3. Adaptive layout portrait (<840 dp) → single-pane (no side-by-side detail)
/// 4. Trading flow — route cards render from mocked data; tapping selects detail
/// 5. Sec-zone filter — toggling Lo/Null changes the client-side filtered list
/// 6. Character flow — Character tab shows name / ship / skills from mocked data
///
/// ## What is NOT automated (manual-on-device coverage)
/// - Real EVE SSO login (interactive browser + OAuth callback)
/// - Real route calculation against a live backend
/// - `flutter_web_auth_2` deep-link handling
/// - Network error / retry paths (require a real Dio stack with 401 replay)
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/features/trading/hub_comparison_screen.dart';
import 'package:eve_o_provit/features/trading/route_detail.dart';
import 'package:eve_o_provit/features/trading/route_list.dart';
import 'package:eve_o_provit/features/trading/trading_screen.dart';
import 'package:eve_o_provit/main.dart';

import 'support/fakes.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Sets the virtual screen size and registers a tear-down to reset it.
void _setViewSize(WidgetTester tester, double width, double height) {
  tester.view.physicalSize = Size(width, height);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
}

/// Pumps the full [EveOProvitApp] (with [goRouterProvider]) using the supplied
/// [overrides] and settles all pending frames.
///
/// [overrides] is typed as `List<dynamic>` because Riverpod's [Override] is
/// not publicly exported by `flutter_riverpod`; the overrides are still typed
/// at the call sites via Riverpod's typed `overrideWith` / `overrideWithValue`
/// helpers.
Future<void> _pumpApp(
  WidgetTester tester,
  List<dynamic> overrides,
) async {
  await tester.pumpWidget(
    ProviderScope(
      // cast needed: Override is sealed/unexported in flutter_riverpod v3
      overrides: overrides.cast(),
      child: const EveOProvitApp(),
    ),
  );
  // Multiple pumps let go_router evaluate its redirect after the async auth
  // notifier resolves (AsyncLoading → AsyncData).
  await tester.pumpAndSettle();
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  // ── 1. Auth gate ────────────────────────────────────────────────────────────

  group('Auth gate', () {
    testWidgets(
        'Unauthenticated state → LoginScreen shows "Login mit EVE"',
        (tester) async {
      _setViewSize(tester, 800, 1200);

      await _pumpApp(tester, unauthenticatedOverrides());

      expect(find.text('Login mit EVE'), findsOneWidget);
      // Must NOT land on the trading screen.
      expect(find.byType(TradingScreen), findsNothing);
    });

    testWidgets(
        'Authenticated state → app navigates to TradingScreen (not login)',
        (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      // Trading app bar title
      expect(find.text('Trading'), findsWidgets);
      // The "Login mit EVE" button must NOT be present.
      expect(find.text('Login mit EVE'), findsNothing);
    });
  });

  // ── 2 & 3. Adaptive layout ──────────────────────────────────────────────────

  group('Adaptive layout', () {
    testWidgets(
        'Landscape (1280×800, ≥840 dp) → two-pane: RouteList + at least one VerticalDivider',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      expect(find.byType(RouteList), findsOneWidget);
      // VerticalDivider is the structural marker for the multi-column layout.
      // The trading screen now has a sidebar + list + detail = 2 dividers.
      expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
    });

    testWidgets(
        'Portrait (800×1280, <840 dp) → single-pane: RouteList, no VerticalDivider',
        (tester) async {
      _setViewSize(tester, 800, 1280);
      await _pumpApp(tester, authenticatedOverrides());

      expect(find.byType(RouteList), findsOneWidget);
      // In single-pane there is no side-by-side divider.
      expect(find.byType(VerticalDivider), findsNothing);
      // RouteDetail is not embedded — only accessible via push navigation.
      expect(find.byType(RouteDetail), findsNothing);
    });
  });

  // ── 4. Trading flow ─────────────────────────────────────────────────────────

  group('Trading flow', () {
    testWidgets(
        'Route cards render with mocked data (Tritanium, Pyerite visible)',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      // The fake RoutesNotifier seeds routes immediately (no calculate() needed).
      expect(find.text('Tritanium'), findsOneWidget);
      expect(find.text('Pyerite'), findsOneWidget);
      // Low-sec route is hidden by the default high-sec-only filter.
      expect(find.text('Megacyte'), findsNothing);
    });

    testWidgets(
        'Tapping a route card in two-pane selects it and shows RouteDetail',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      // Tap the Tritanium route card.
      await tester.tap(find.text('Tritanium'));
      await tester.pumpAndSettle();

      // The detail pane should now be visible.
      expect(find.byType(RouteDetail), findsOneWidget);
      expect(find.text('Handelsroute'), findsOneWidget);
    });

    testWidgets(
        'Berechnen button is present; routes from stub are shown immediately',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      // The calculate button must be reachable in the controls bar.
      expect(find.text('Berechnen'), findsOneWidget);

      // FakeRoutesNotifier seeds routes immediately — no interaction required.
      // This verifies the full rendering pipeline: auth gate → shell →
      // TradingScreen → ControlsBar + RouteList.
      expect(find.text('Tritanium'), findsOneWidget);
      expect(find.text('Pyerite'), findsOneWidget);
    });
  });

  // ── 5. Sec-zone filter ──────────────────────────────────────────────────────

  group('Sec-zone filter', () {
    testWidgets(
        'Default (high-sec only): low-sec route hidden; toggling "Lo" reveals it',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      // Megacyte (low-sec) is hidden by default.
      expect(find.text('Megacyte'), findsNothing);
      expect(find.text('Tritanium'), findsOneWidget);

      // Toggle the "Lo" sec-zone pill.
      await tester.tap(find.text('Low Sec (< 0.5)'));
      await tester.pumpAndSettle();

      // Now the low-sec route appears.
      expect(find.text('Megacyte'), findsOneWidget);
      // High-sec routes remain.
      expect(find.text('Tritanium'), findsOneWidget);
    });

    testWidgets('Toggling "Lo" off again hides the low-sec route',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      // Enable low-sec.
      await tester.tap(find.text('Low Sec (< 0.5)'));
      await tester.pumpAndSettle();
      expect(find.text('Megacyte'), findsOneWidget);

      // Disable it again.
      await tester.tap(find.text('Low Sec (< 0.5)'));
      await tester.pumpAndSettle();
      expect(find.text('Megacyte'), findsNothing);
    });

    testWidgets('"Null" toggle does not affect high-/low-sec routes',
        (tester) async {
      _setViewSize(tester, 1280, 800);
      await _pumpApp(tester, authenticatedOverrides());

      // Canned data has no null-sec routes; toggling Null must not change the list.
      expect(find.text('Tritanium'), findsOneWidget);
      await tester.tap(find.text('Null Sec (≤ 0.0)'));
      await tester.pumpAndSettle();
      // High-sec routes still present.
      expect(find.text('Tritanium'), findsOneWidget);
      // No null-sec routes in canned data, so nothing extra appears.
      expect(find.text('Megacyte'), findsNothing);
    });
  });

  // ── 6. Character flow ───────────────────────────────────────────────────────

  group('Character flow', () {
    // authenticatedOverrides() now includes characterApiProvider fake;
    // no need to extend — adding it again would cause a duplicate-override error.

    testWidgets(
        'Navigating to Character tab shows character name and ship from mocked data',
        (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      // Tap the "Character" bottom navigation destination.
      await tester.tap(find.text('Character').last);
      await tester.pumpAndSettle();

      // Character name from fakeCharacterInfo.
      expect(find.text('Test Pilot'), findsOneWidget);

      // Active ship section (card title).
      expect(find.text('Aktives Schiff'), findsOneWidget);

      // Ship name from fakeCharacterShip.
      expect(find.text('My Badger'), findsOneWidget);
    });

    testWidgets(
        'Character tab shows skills section with Navigation skill visible',
        (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      await tester.tap(find.text('Character').last);
      await tester.pumpAndSettle();

      // The skills card section header.
      expect(find.text('Skills'), findsOneWidget);

      // 'Navigation' is in fakeCharacterSkills and is a displayed skill.
      expect(find.text('Navigation'), findsOneWidget);
    });

    testWidgets(
        'Character tab portrait (800×1280) still shows name and ship',
        (tester) async {
      _setViewSize(tester, 800, 1280);

      await _pumpApp(tester, authenticatedOverrides());

      await tester.tap(find.text('Character').last);
      await tester.pumpAndSettle();

      expect(find.text('Test Pilot'), findsOneWidget);
      expect(find.text('My Badger'), findsOneWidget);
    });
  });

  // ── 6b. Hub Comparison ────────────────────────────────────────────────────

  group('Hub Comparison', () {
    testWidgets(
        'Navigating to Hub-Vergleich tab renders the comparison table from '
        'mocked data (hub names + recommended hub)', (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      // Open the Hub-Vergleich bottom-nav destination.
      await tester.tap(find.text('Hub-Vergleich').last);
      await tester.pumpAndSettle();

      expect(find.byType(HubComparisonScreen), findsOneWidget);
      // Item name header from fakeHubResponse.
      expect(find.text('Tritanium'), findsWidgets);
      // Hub names render.
      expect(find.text('Dodixie'), findsOneWidget);
      expect(find.text('Jita'), findsOneWidget);
      expect(find.text('Rens'), findsOneWidget);
      // Recommended hub (Dodixie, best_hub_region_id) shows the star.
      expect(find.byIcon(Icons.star_rounded), findsOneWidget);
      // No-data hub placeholder.
      expect(find.text('keine Daten'), findsOneWidget);
    });

    testWidgets(
        'Hub Comparison portrait (800×1280) is single-pane (no VerticalDivider)',
        (tester) async {
      _setViewSize(tester, 800, 1280);

      await _pumpApp(tester, authenticatedOverrides());

      await tester.tap(find.text('Hub-Vergleich').last);
      await tester.pumpAndSettle();

      expect(find.byType(HubComparisonScreen), findsOneWidget);
      expect(find.byType(VerticalDivider), findsNothing);
      expect(find.text('Dodixie'), findsOneWidget);
    });
  });

  // ── 7. Logout flow ──────────────────────────────────────────────────────────

  group('Logout flow', () {
    // The logout "Abmelden" destination lives in the bottom NavigationBar next
    // to Trading/Character (not hidden in a screen AppBar). Tapping it opens a
    // confirm dialog; it is an action, not a navigation target.

    testWidgets(
        'Tapping "Abmelden" opens the confirm dialog without changing the tab',
        (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      // Starts on Trading.
      expect(find.text('Trading'), findsWidgets);

      // Tap the bottom-bar "Abmelden" destination (only the nav label so far).
      await tester.tap(find.text('Abmelden'));
      await tester.pumpAndSettle();

      // Confirm dialog appears; still authenticated (login button absent).
      expect(find.byType(AlertDialog), findsOneWidget);
      expect(find.text('Abmelden?'), findsOneWidget);
      expect(find.text('Login mit EVE'), findsNothing);
    });

    testWidgets(
        'Confirming logout clears the session and redirects to LoginScreen',
        (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      await tester.tap(find.text('Abmelden'));
      await tester.pumpAndSettle();

      // Tap the dialog's "Abmelden" confirm button (disambiguated from the
      // nav-bar label by scoping to the AlertDialog).
      await tester.tap(
        find.descendant(
          of: find.byType(AlertDialog),
          matching: find.text('Abmelden'),
        ),
      );
      await tester.pumpAndSettle();

      // Auth state → Unauthenticated → router redirects to the login screen.
      expect(find.text('Login mit EVE'), findsOneWidget);
      expect(find.byType(TradingScreen), findsNothing);
    });

    testWidgets(
        'Cancelling the dialog keeps the user logged in on Trading',
        (tester) async {
      _setViewSize(tester, 1280, 800);

      await _pumpApp(tester, authenticatedOverrides());

      await tester.tap(find.text('Abmelden'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Abbrechen'));
      await tester.pumpAndSettle();

      // Dialog dismissed, still authenticated and on the trading screen.
      expect(find.byType(AlertDialog), findsNothing);
      expect(find.byType(TradingScreen), findsOneWidget);
      expect(find.text('Login mit EVE'), findsNothing);
    });
  });
}

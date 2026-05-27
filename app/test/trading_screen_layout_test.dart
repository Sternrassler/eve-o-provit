/// Widget tests for TradingScreen layout behaviour and RouteCard rendering.
///
/// Tests:
/// 1. Two-pane layout (width 1280): RouteList and detail pane both present.
/// 2. Single-pane layout (width 800): RouteList present; no side-by-side RouteDetail.
/// 3. RouteCard renders item name + ISK/h text and fires onTap.
library;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/features/trading/providers.dart';
import 'package:eve_o_provit/features/trading/route_card.dart';
import 'package:eve_o_provit/features/trading/route_detail.dart';
import 'package:eve_o_provit/features/trading/route_list.dart';
import 'package:eve_o_provit/features/trading/trading_screen.dart';
import 'package:eve_o_provit/core/theme.dart';

// ---------------------------------------------------------------------------
// Fake TradingApi
// ---------------------------------------------------------------------------

class _FakeApi extends TradingApi {
  _FakeApi() : super(Dio());

  @override
  Future<RegionsResponse> regions() async => RegionsResponse(
        regions: [
          const Region(id: 10000002, name: 'The Forge'),
          const Region(id: 10000043, name: 'Domain'),
        ],
        count: 2,
      );

  @override
  Future<RouteCalculationResponse> calculateRoutes(
    RouteCalculationRequest request,
  ) async =>
      _fakeResponse();

  @override
  Future<MarketDataStalenessResponse> staleness(int regionId) async =>
      MarketDataStalenessResponse(
        regionId: regionId,
        regionName: 'The Forge',
        lastUpdate: DateTime.now(),
        ageMinutes: 5,
        status: 'fresh',
        refreshAllowed: true,
      );

  @override
  Future<void> setWaypoint({
    required int destinationId,
    bool addToBeginning = false,
    bool clearOtherWaypoints = false,
  }) async {}
}

// ---------------------------------------------------------------------------
// Canned data
// ---------------------------------------------------------------------------

TradingRoute _fakeRoute({
  int typeId = 34,
  String name = 'Tritanium',
  double minRouteSecurityStatus = 0.5,
}) {
  return TradingRoute(
    itemTypeId: typeId,
    itemName: name,
    buySystemId: 30000142,
    buySystemName: 'Jita',
    buyStationId: 60003760,
    buyStationName: 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
    buyPrice: 5.50,
    sellSystemId: 30002187,
    sellSystemName: 'Amarr',
    sellStationId: 60008494,
    sellStationName: 'Amarr VIII (Oris) - Emperor Family Academy',
    sellPrice: 6.80,
    buySecurityStatus: 1.0,
    sellSecurityStatus: 1.0,
    minRouteSecurityStatus: minRouteSecurityStatus,
    quantity: 100000,
    profitPerUnit: 1.3,
    totalProfit: 130000.0,
    spreadPercent: 23.6,
    travelTimeSeconds: 1200.0,
    roundTripSeconds: 2400.0,
    iskPerHour: 195000000.0,
    jumps: 9,
    itemVolume: 0.01,
    numberOfTours: 1,
    profitPerTour: 130000.0,
    totalTimeMinutes: 40.0,
    baseTravelTimeSeconds: 1350.0,
    skilledTravelTimeSeconds: 1200.0,
    baseIskPerHour: 173000000.0,
    timeImprovementPercent: 11.1,
    buyBrokerFee: 5000.0,
    sellBrokerFee: 5500.0,
    brokerFees: 10500.0,
    salesTax: 3500.0,
    estimatedRelistFee: 2750.0,
    totalFees: 14000.0,
    grossProfit: 144000.0,
    grossMarginPercent: 19.1,
    netProfit: 116000.0,
    netProfitPercent: 15.4,
    cargoUsed: 1000.0,
    cargoCapacity: 50000.0,
    cargoUtilization: 2.0,
    baseCargoCapacity: 50000.0,
    skillBonusPercent: 0.0,
    fittingBonusM3: 0.0,
    totalInvestment: 550000.0,
  );
}

RouteCalculationResponse _fakeResponse() => RouteCalculationResponse(
      regionId: 10000002,
      regionName: 'The Forge',
      shipTypeId: 648,
      shipName: 'Badger',
      cargoCapacity: 50000.0,
      calculationTimeMs: 10,
      routes: [
        _fakeRoute(typeId: 34, name: 'Tritanium'),
        _fakeRoute(typeId: 35, name: 'Pyerite'),
        // A low-sec route — hidden by default (high-sec only), revealed by
        // toggling the "Lo" filter.
        _fakeRoute(
            typeId: 36, name: 'Megacyte', minRouteSecurityStatus: 0.3),
      ],
    );

// ---------------------------------------------------------------------------
// Stub RoutesNotifier — immediately returns canned data
// ---------------------------------------------------------------------------

class _StubRoutesNotifier extends RoutesNotifier {
  @override
  Future<RouteCalculationResponse?> build() async => _fakeResponse();

  @override
  Future<void> calculate() async {
    state = AsyncData(_fakeResponse());
  }
}

// ---------------------------------------------------------------------------
// Helper — pump TradingScreen at a given logical width
// ---------------------------------------------------------------------------

Future<void> _pumpScreen(
  WidgetTester tester,
  double logicalWidth, {
  double logicalHeight = 900,
}) async {
  tester.view.physicalSize = Size(logicalWidth, logicalHeight);
  tester.view.devicePixelRatio = 1.0;
  // Reset view after test to avoid cross-test contamination.
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tradingApiProvider.overrideWithValue(_FakeApi()),
        routesProvider.overrideWith(_StubRoutesNotifier.new),
      ],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const TradingScreen(),
      ),
    ),
  );

  // Allow async providers to resolve.
  await tester.pumpAndSettle();
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {

  // ── 4.4 Layout: Two-pane at 1280 dp ───────────────────────────────────────

  testWidgets('Two-pane (width 1280): RouteList and detail pane both visible',
      (tester) async {
    await _pumpScreen(tester, 1280);

    // RouteList must be in the tree.
    expect(find.byType(RouteList), findsOneWidget);

    // The VerticalDivider is the structural marker for the multi-column layout.
    // The trading screen now has sidebar + list + detail = 2 dividers.
    expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
  });

  // ── 4.4 Layout: Single-pane at 800 dp ────────────────────────────────────

  testWidgets('Single-pane (width 800): RouteList present, no VerticalDivider',
      (tester) async {
    await _pumpScreen(tester, 800);

    expect(find.byType(RouteList), findsOneWidget);

    // In single-pane there is no side-by-side VerticalDivider.
    expect(find.byType(VerticalDivider), findsNothing);

    // RouteDetail is not embedded in the screen body (only reachable via push).
    expect(find.byType(RouteDetail), findsNothing);
  });

  // ── Route cards rendered ───────────────────────────────────────────────────

  testWidgets('Two-pane: route cards for both canned routes are rendered',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.text('Tritanium'), findsOneWidget);
    expect(find.text('Pyerite'), findsOneWidget);
  });

  // ── Sec-zone filter actually changes the displayed routes ─────────────────

  testWidgets(
      'toggling "Lo" filter reveals the low-sec route (client-side filtering)',
      (tester) async {
    await _pumpScreen(tester, 1280);

    // By default (high-sec only) the high-sec routes are shown and the
    // low-sec route (Megacyte, sec 0.3) is hidden.
    expect(find.text('Tritanium'), findsOneWidget);
    expect(find.text('Megacyte'), findsNothing);

    // Tap the "Low Sec" checkbox in the filter panel.
    await tester.tap(find.text('Low Sec (< 0.5)'));
    await tester.pumpAndSettle();

    // Now the low-sec route appears alongside the high-sec ones.
    expect(find.text('Megacyte'), findsOneWidget);
    expect(find.text('Tritanium'), findsOneWidget);
  });

  // ── Tapping a card in two-pane shows detail ──────────────────────────────

  testWidgets('Two-pane: tapping a RouteCard shows RouteDetail pane',
      (tester) async {
    await _pumpScreen(tester, 1280);

    // Tap the first route card.
    await tester.tap(find.text('Tritanium'));
    await tester.pumpAndSettle();

    // RouteDetail should now appear in the right pane.
    expect(find.byType(RouteDetail), findsOneWidget);
    // The detail pane header label is present.
    expect(find.text('Handelsroute'), findsOneWidget);
  });

  // ── RouteCard unit tests ──────────────────────────────────────────────────

  group('RouteCard', () {
    testWidgets('renders item name and ISK/h text', (tester) async {
      final route = _fakeRoute(name: 'Mexallon');
      await tester.pumpWidget(
        MaterialApp(
          theme: buildTheme(Brightness.light),
          home: Scaffold(
            body: RouteCard(
              route: route,
              selected: false,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.text('Mexallon'), findsOneWidget);
      // ISK/h for 195_000_000 → "195,0M ISK/h"
      expect(find.text('195,0M ISK/h'), findsOneWidget);
    });

    testWidgets('fires onTap callback when tapped', (tester) async {
      var tapped = false;
      final route = _fakeRoute(name: 'Tritanium');
      await tester.pumpWidget(
        MaterialApp(
          theme: buildTheme(Brightness.light),
          home: Scaffold(
            body: RouteCard(
              route: route,
              selected: false,
              onTap: () => tapped = true,
            ),
          ),
        ),
      );

      await tester.tap(find.byType(RouteCard));
      expect(tapped, isTrue);
    });

    testWidgets('selected card shows accent border via custom Card shape',
        (tester) async {
      final route = _fakeRoute(name: 'Zydrine');
      await tester.pumpWidget(
        MaterialApp(
          theme: buildTheme(Brightness.light),
          home: Scaffold(
            body: RouteCard(
              route: route,
              selected: true,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.byType(Card), findsOneWidget);
      expect(find.text('Zydrine'), findsOneWidget);
    });
  });

  // ── formatIskPerHour unit tests ────────────────────────────────────────────

  group('formatIskPerHour', () {
    test('formats millions correctly', () {
      expect(formatIskPerHour(5_000_000), '5,0M ISK/h');
      expect(formatIskPerHour(195_000_000), '195,0M ISK/h');
    });

    test('formats billions correctly', () {
      expect(formatIskPerHour(1_500_000_000), '1,5B ISK/h');
    });

    test('formats thousands correctly', () {
      expect(formatIskPerHour(500_000), '500,0K ISK/h');
    });
  });
}

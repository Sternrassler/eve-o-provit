/// Behavioral tests for the trading Riverpod providers.
///
/// Uses [ProviderContainer] + a fake [TradingApi] (via tradingApiProvider
/// override) so no network calls are made.
library;

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart'
    show currentShipProvider;
import 'package:eve_o_provit/features/trading/providers.dart';

// ---------------------------------------------------------------------------
// Fake TradingApi — driven by injected callbacks so each test controls data
// ---------------------------------------------------------------------------

/// A minimal [TradingApi] subclass that delegates to provided callbacks,
/// letting each test inject exactly the data it needs.
class FakeTradingApi extends TradingApi {
  FakeTradingApi({
    required Future<RegionsResponse> Function() regionsCallback,
    Future<RouteCalculationResponse> Function(RouteCalculationRequest)?
        calculateCallback,
  })  : _regionsCallback = regionsCallback,
        _calculateCallback = calculateCallback,
        super(Dio()); // Dio instance is never used; calls are intercepted.

  final Future<RegionsResponse> Function() _regionsCallback;
  final Future<RouteCalculationResponse> Function(RouteCalculationRequest)?
      _calculateCallback;

  @override
  Future<RegionsResponse> regions() => _regionsCallback();

  @override
  Future<RouteCalculationResponse> calculateRoutes(
    RouteCalculationRequest request,
  ) =>
      _calculateCallback != null
          ? _calculateCallback(request)
          : Future.error(StateError('calculateCallback not provided'));
}

// ---------------------------------------------------------------------------
// Shared fixtures
// ---------------------------------------------------------------------------

final _fakeRegions = [
  const Region(id: 10000002, name: 'The Forge'),
  const Region(id: 10000043, name: 'Domain'),
];

TradingRoute _fakeRoute({
  int typeId = 34,
  String name = 'Tritanium',
  double minRouteSecurityStatus = 0.5,
  double spreadPercent = 23.6,
  double netProfit = 116000.0,
  double totalProfit = 130000.0,
  double travelTimeSeconds = 1200.0, // 20 min — within default 30-min limit
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
    totalProfit: totalProfit,
    spreadPercent: spreadPercent,
    travelTimeSeconds: travelTimeSeconds,
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
    netProfit: netProfit,
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

RouteCalculationResponse _fakeResponse({
  List<TradingRoute>? routes,
  String? warning,
}) {
  return RouteCalculationResponse(
    regionId: 10000002,
    regionName: 'The Forge',
    shipTypeId: 649,
    shipName: 'Badger',
    cargoCapacity: 50000.0,
    calculationTimeMs: 42,
    routes: routes ?? [_fakeRoute()],
    warning: warning,
  );
}

/// A [CharacterShip] fixture for the current ship (the ship is no longer
/// user-selected; [currentShipProvider] is the source of truth). [effectiveM3]
/// (when > 0 and not [unavailable]) becomes the `cargo_capacity` override.
CharacterShip _ship({
  int typeId = 649,
  double? effectiveM3,
  bool unavailable = false,
}) =>
    CharacterShip(
      shipTypeId: typeId,
      shipName: 'Test Ship',
      shipItemId: 1,
      shipTypeName: 'Test',
      cargoCapacity: 5000,
      effectiveCargoCapacity: effectiveM3,
      effectiveCargoUnavailable: unavailable,
    );

// ---------------------------------------------------------------------------
// Helper — builds a ProviderContainer with tradingApiProvider overridden and
// an optional current-ship override.
// ---------------------------------------------------------------------------

ProviderContainer _makeContainer(FakeTradingApi fakeApi, {CharacterShip? ship}) {
  return ProviderContainer(
    overrides: [
      tradingApiProvider.overrideWithValue(fakeApi),
      if (ship != null)
        currentShipProvider.overrideWith((ref) async => ship),
    ],
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  // ── regionsProvider ────────────────────────────────────────────────────────

  group('regionsProvider', () {
    test('resolves to the list of regions returned by the api', () async {
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: _fakeRegions.length),
      );
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      final result = await container.read(regionsProvider.future);

      expect(result, hasLength(2));
      expect(result[0].id, 10000002);
      expect(result[0].name, 'The Forge');
      expect(result[1].name, 'Domain');
    });

    test('returns different regions for different fake data (isolation check)',
        () async {
      // Verify that the provider correctly forwards whatever regions the api
      // returns — a second container with different data should differ.
      final api2 = FakeTradingApi(
        regionsCallback: () async => RegionsResponse(
          regions: [const Region(id: 10000033, name: 'The Citadel')],
          count: 1,
        ),
      );
      final container2 = ProviderContainer(
        overrides: [tradingApiProvider.overrideWithValue(api2)],
      );
      addTearDown(container2.dispose);

      final result = await container2.read(regionsProvider.future);

      expect(result, hasLength(1));
      expect(result[0].name, 'The Citadel');
    });
  });

  // ── routesProvider — successful calculation ────────────────────────────────

  group('routesProvider.calculate — success', () {
    test('build() starts with null (no calculation yet)', () async {
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
      );
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      final value = await container.read(routesProvider.future);
      expect(value, isNull);
    });

    test('calculate() populates state with the api response', () async {
      final expectedResponse = _fakeResponse();
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (_) async => expectedResponse,
      );
      final container = _makeContainer(api, ship: _ship(typeId: 649));
      addTearDown(container.dispose);

      // Set up required selection state (region) — ship comes from the override.
      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      // Ensure the current ship is resolved before calculating.
      await container.read(currentShipProvider.future);

      // Trigger calculation.
      await container.read(routesProvider.notifier).calculate();

      final result = await container.read(routesProvider.future);
      expect(result, isNotNull);
      expect(result!.routes, hasLength(1));
      expect(result.routes[0].itemName, 'Tritanium');
      expect(result.routes[0].iskPerHour, 195000000.0);
    });

    test('calculate() passes region and shipTypeId from selection providers',
        () async {
      RouteCalculationRequest? capturedRequest;
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (req) async {
          capturedRequest = req;
          return _fakeResponse();
        },
      );
      final container = _makeContainer(api, ship: _ship(typeId: 12345));
      addTearDown(container.dispose);

      container.read(selectedRegionProvider.notifier).state = _fakeRegions[1];
      await container.read(currentShipProvider.future);

      await container.read(routesProvider.notifier).calculate();

      expect(capturedRequest?.regionId, 10000043); // Domain
      expect(capturedRequest?.shipTypeId, 12345);
    });

    test(
        'calculate() sends the current ship effective cargo as cargo_capacity '
        'override (omitted when unavailable/0)', () async {
      RouteCalculationRequest? capturedRequest;
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (req) async {
          capturedRequest = req;
          return _fakeResponse();
        },
      );

      // 1) effective cargo known → sent as cargo_capacity.
      final c1 = _makeContainer(
        api,
        ship: _ship(typeId: 648, effectiveM3: 9656.0),
      );
      addTearDown(c1.dispose);
      c1.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      await c1.read(currentShipProvider.future);
      await c1.read(routesProvider.notifier).calculate();
      expect(capturedRequest?.cargoCapacity, 9656.0);
      expect(capturedRequest!.toJson()['cargo_capacity'], 9656.0);

      // 2) effective cargo unavailable → override omitted.
      capturedRequest = null;
      final c2 = _makeContainer(
        api,
        ship: _ship(typeId: 648, effectiveM3: 9656.0, unavailable: true),
      );
      addTearDown(c2.dispose);
      c2.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      await c2.read(currentShipProvider.future);
      await c2.read(routesProvider.notifier).calculate();
      expect(capturedRequest?.cargoCapacity, isNull);
      expect(capturedRequest!.toJson().containsKey('cargo_capacity'), isFalse);
    });

    test(
        'calculate() sends ONLY region_id + ship_type_id — no security/filter '
        'params (backend has none; filtering is client-side)', () async {
      RouteCalculationRequest? capturedRequest;
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (req) async {
          capturedRequest = req;
          return _fakeResponse();
        },
      );
      // Ship with NO effective cargo → no cargo_capacity override, so the body
      // carries only region_id + ship_type_id.
      final container = _makeContainer(api, ship: _ship(typeId: 648));
      addTearDown(container.dispose);

      // Set non-default filters; they must NOT leak into the request body.
      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(
              lowSec: true,
              nullSec: true,
              minSpread: 10,
              minProfit: 250000,
            ),
          );
      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      await container.read(currentShipProvider.future);

      await container.read(routesProvider.notifier).calculate();

      // The request carries only the two required fields. The previous bug
      // mapped filters.minProfit → min_daily_volume; that must be gone.
      expect(capturedRequest?.minDailyVolume, isNull);
      final json = capturedRequest!.toJson();
      expect(json.keys, unorderedEquals(<String>['region_id', 'ship_type_id']));
    });
  });

  // ── routesProvider — empty routes + warning ────────────────────────────────

  group('routesProvider.calculate — empty routes', () {
    test('empty routes list is a success state, not an error', () async {
      final emptyResponse = _fakeResponse(
        routes: [],
        warning: 'No profitable routes found for selected parameters',
      );
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (_) async => emptyResponse,
      );
      final container = _makeContainer(api, ship: _ship(typeId: 649));
      addTearDown(container.dispose);

      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      await container.read(currentShipProvider.future);

      await container.read(routesProvider.notifier).calculate();

      // State must be AsyncData — not AsyncError.
      final asyncValue = container.read(routesProvider);
      expect(asyncValue, isA<AsyncData<RouteCalculationResponse?>>());

      final result = asyncValue.value!;
      expect(result.routes, isEmpty);
      expect(result.warning, isNotNull);
      expect(
        result.warning,
        contains('No profitable routes'),
      );
    });
  });

  // ── routesProvider — guards when selection is incomplete ──────────────────

  group('routesProvider.calculate — incomplete selection', () {
    test('calculate() is a no-op when region is null', () async {
      bool apiCalled = false;
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (_) async {
          apiCalled = true;
          return _fakeResponse();
        },
      );
      // ship set, but region remains null.
      final container = _makeContainer(api, ship: _ship(typeId: 649));
      addTearDown(container.dispose);
      await container.read(currentShipProvider.future);

      await container.read(routesProvider.notifier).calculate();

      expect(apiCalled, isFalse);
      // State should still be null (unchanged from build).
      final result = await container.read(routesProvider.future);
      expect(result, isNull);
    });

    test('calculate() is a no-op when the current ship is not loaded (null)',
        () async {
      bool apiCalled = false;
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (_) async {
          apiCalled = true;
          return _fakeResponse();
        },
      );
      // currentShipProvider resolves to null (no current ship loaded).
      final container = ProviderContainer(
        overrides: [
          tradingApiProvider.overrideWithValue(api),
          currentShipProvider.overrideWith((ref) async => null),
        ],
      );
      addTearDown(container.dispose);

      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      await container.read(currentShipProvider.future);

      await container.read(routesProvider.notifier).calculate();

      expect(apiCalled, isFalse);
      final result = await container.read(routesProvider.future);
      expect(result, isNull);
    });
  });

  // ── filtersProvider ────────────────────────────────────────────────────────

  group('filtersProvider', () {
    test(
        'has correct web-matching defaults: '
        'highSec=true, lowSec=false, nullSec=false, '
        'minSpread=5, minProfit=100000, maxTravelTime=30',
        () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final filters = container.read(filtersProvider);

      expect(filters.highSec, isTrue);
      expect(filters.lowSec, isFalse);
      expect(filters.nullSec, isFalse);
      expect(filters.minSpread, 5.0);
      expect(filters.minProfit, 100000.0);
      expect(filters.maxTravelTime, 30.0);
    });

    test('copyWith toggles lowSec without affecting other fields', () {
      const original = TradingFilters(highSec: true, lowSec: false);

      final updated = original.copyWith(lowSec: true);

      expect(updated.lowSec, isTrue);
      expect(updated.highSec, isTrue); // unchanged
      expect(updated.nullSec, isFalse); // unchanged
    });

    test('copyWith toggles nullSec independently', () {
      const original = TradingFilters();

      final updated = original.copyWith(nullSec: true);

      expect(updated.nullSec, isTrue);
      expect(updated.highSec, isTrue); // unchanged
      expect(updated.lowSec, isFalse); // unchanged
    });

    test('update() via notifier replaces state immutably', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(lowSec: true, nullSec: true),
          );

      final updated = container.read(filtersProvider);
      expect(updated.lowSec, isTrue);
      expect(updated.nullSec, isTrue);
      expect(updated.highSec, isTrue); // default preserved
    });

    test('all three security zones can be toggled independently', () {
      const base = TradingFilters(highSec: true, lowSec: false, nullSec: false);

      final allOn = base
          .copyWith(highSec: true, lowSec: true, nullSec: true);
      expect(allOn.highSec, isTrue);
      expect(allOn.lowSec, isTrue);
      expect(allOn.nullSec, isTrue);

      final allOff =
          base.copyWith(highSec: false, lowSec: false, nullSec: false);
      expect(allOff.highSec, isFalse);
      expect(allOff.lowSec, isFalse);
      expect(allOff.nullSec, isFalse);
    });
  });

  // ── selectedRouteProvider ─────────────────────────────────────────────────

  group('selectedRouteProvider', () {
    test('starts as null', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      expect(container.read(selectedRouteProvider), isNull);
    });

    test('can be set to a TradingRoute and read back', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final route = _fakeRoute(typeId: 35, name: 'Pyerite');
      container.read(selectedRouteProvider.notifier).state = route;

      final selected = container.read(selectedRouteProvider);
      expect(selected?.itemName, 'Pyerite');
      expect(selected?.itemTypeId, 35);
    });

    test('can be cleared back to null', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(selectedRouteProvider.notifier).state = _fakeRoute();
      container.read(selectedRouteProvider.notifier).state = null;

      expect(container.read(selectedRouteProvider), isNull);
    });
  });

  // ── filteredRoutesProvider — client-side security-zone filtering ──────────

  group('filteredRoutesProvider (client-side filtering)', () {
    // Build a container with a calculation already performed, returning the
    // three supplied routes.
    Future<ProviderContainer> containerWithRoutes(
      List<TradingRoute> routes,
    ) async {
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (_) async => _fakeResponse(routes: routes),
      );
      final container = _makeContainer(api, ship: _ship(typeId: 648));
      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      await container.read(currentShipProvider.future);
      await container.read(routesProvider.notifier).calculate();
      return container;
    }

    final hiSec = _fakeRoute(
        typeId: 1, name: 'HiSec', minRouteSecurityStatus: 0.7);
    final lowSec = _fakeRoute(
        typeId: 2, name: 'LowSec', minRouteSecurityStatus: 0.3);
    final nullSec = _fakeRoute(
        typeId: 3, name: 'NullSec', minRouteSecurityStatus: -0.1);

    test('default filters (high-sec only) hide low/null routes', () async {
      final container = await containerWithRoutes([hiSec, lowSec, nullSec]);
      addTearDown(container.dispose);

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HiSec']);
    });

    test('enabling low-sec reveals low-sec routes', () async {
      final container = await containerWithRoutes([hiSec, lowSec, nullSec]);
      addTearDown(container.dispose);

      // Sanity: before toggling, only HiSec is shown.
      expect(container.read(filteredRoutesProvider).map((r) => r.itemName),
          ['HiSec']);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(lowSec: true),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HiSec', 'LowSec']);
    });

    test('enabling null-sec reveals null-sec routes', () async {
      final container = await containerWithRoutes([hiSec, lowSec, nullSec]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(lowSec: true, nullSec: true),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(
        filtered.map((r) => r.itemName),
        ['HiSec', 'LowSec', 'NullSec'],
      );
    });

    test('disabling high-sec hides high-sec routes', () async {
      final container = await containerWithRoutes([hiSec, lowSec, nullSec]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(highSec: false, lowSec: true),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['LowSec']);
    });

    test('sec=0.0 boundary is classified as null-sec (sec <= 0.0)', () async {
      final atZero = _fakeRoute(
          typeId: 9, name: 'AtZero', minRouteSecurityStatus: 0.0);
      final container = await containerWithRoutes([atZero]);
      addTearDown(container.dispose);

      // High-sec only (default) → hidden.
      expect(container.read(filteredRoutesProvider), isEmpty);

      // Null-sec on → visible.
      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(nullSec: true),
          );
      expect(container.read(filteredRoutesProvider).map((r) => r.itemName),
          ['AtZero']);
    });

    test(
        'default minSpread=5 hides routes with spreadPercent < 5',
        () async {
      // Default filters include minSpread=5; spread=2 is below threshold.
      final lowSpread =
          _fakeRoute(typeId: 4, name: 'LowSpread', spreadPercent: 2);
      final highSpread =
          _fakeRoute(typeId: 5, name: 'HighSpread', spreadPercent: 20);
      final container = await containerWithRoutes([lowSpread, highSpread]);
      addTearDown(container.dispose);

      // No manual update needed — default minSpread=5 already hides LowSpread.
      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HighSpread']);
    });

    test('raising minSpread to 15 hides medium-spread routes', () async {
      final midSpread =
          _fakeRoute(typeId: 4, name: 'MidSpread', spreadPercent: 10);
      final highSpread =
          _fakeRoute(typeId: 5, name: 'HighSpread', spreadPercent: 20);
      final container = await containerWithRoutes([midSpread, highSpread]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(minSpread: 15),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HighSpread']);
    });

    test(
        'default minProfit=100000 hides routes with totalProfit < 100000',
        () async {
      // The web predicate filters on totalProfit (not netProfit).
      final lowProfit =
          _fakeRoute(typeId: 6, name: 'LowProfit', totalProfit: 50000);
      final highProfit =
          _fakeRoute(typeId: 7, name: 'HighProfit', totalProfit: 200000);
      final container = await containerWithRoutes([lowProfit, highProfit]);
      addTearDown(container.dispose);

      // Default minProfit=100000 already hides LowProfit.
      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HighProfit']);
    });

    test('raising minProfit to 300000 hides medium-profit routes', () async {
      final midProfit =
          _fakeRoute(typeId: 6, name: 'MidProfit', totalProfit: 150000);
      final highProfit =
          _fakeRoute(typeId: 7, name: 'HighProfit', totalProfit: 500000);
      final container = await containerWithRoutes([midProfit, highProfit]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(minProfit: 300000),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HighProfit']);
    });

    test('routes with netProfit < 0 are always dropped (web predicate)', () async {
      final negative =
          _fakeRoute(typeId: 8, name: 'Negative', netProfit: -500);
      final positive =
          _fakeRoute(typeId: 9, name: 'Positive', netProfit: 50000);
      // Enable all sec-zones and set very permissive filters so only netProfit
      // check is the discriminator.
      final container = await containerWithRoutes([negative, positive]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(
              minSpread: 0,
              minProfit: 0,
              maxTravelTime: 60,
            ),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['Positive']);
    });

    test('maxTravelTime=30 hides routes that take more than 30 min', () async {
      // 1200 s = 20 min → passes; 2400 s = 40 min → filtered out.
      final fast =
          _fakeRoute(typeId: 10, name: 'Fast', travelTimeSeconds: 1200);
      final slow =
          _fakeRoute(typeId: 11, name: 'Slow', travelTimeSeconds: 2400);
      final container = await containerWithRoutes([fast, slow]);
      addTearDown(container.dispose);

      // Default maxTravelTime=30 min already hides Slow.
      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['Fast']);
    });

    test('raising maxTravelTime to 60 reveals slow routes', () async {
      final fast =
          _fakeRoute(typeId: 10, name: 'Fast', travelTimeSeconds: 1200);
      final slow =
          _fakeRoute(typeId: 11, name: 'Slow', travelTimeSeconds: 2400);
      final container = await containerWithRoutes([fast, slow]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(maxTravelTime: 60),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['Fast', 'Slow']);
    });

    test('copyWith preserves maxTravelTime when only sec-zone is toggled', () {
      const f = TradingFilters();
      expect(f.maxTravelTime, 30.0);

      final updated = f.copyWith(lowSec: true);
      expect(updated.maxTravelTime, 30.0); // unchanged
    });

    test('returns empty list before any calculation', () {
      final container = ProviderContainer(
        overrides: [
          tradingApiProvider.overrideWithValue(
            FakeTradingApi(
              regionsCallback: () async =>
                  RegionsResponse(regions: _fakeRegions, count: 2),
            ),
          ),
        ],
      );
      addTearDown(container.dispose);

      expect(container.read(filteredRoutesProvider), isEmpty);
    });
  });
}

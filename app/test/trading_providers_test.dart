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
    spreadPercent: spreadPercent,
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

// ---------------------------------------------------------------------------
// Helper — builds a ProviderContainer with tradingApiProvider overridden
// ---------------------------------------------------------------------------

ProviderContainer _makeContainer(FakeTradingApi fakeApi) {
  return ProviderContainer(
    overrides: [
      tradingApiProvider.overrideWithValue(fakeApi),
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
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      // Set up required selection state.
      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      container.read(selectedShipTypeIdProvider.notifier).state = 649;

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
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      container.read(selectedRegionProvider.notifier).state = _fakeRegions[1];
      container.read(selectedShipTypeIdProvider.notifier).state = 12345;

      await container.read(routesProvider.notifier).calculate();

      expect(capturedRequest?.regionId, 10000043); // Domain
      expect(capturedRequest?.shipTypeId, 12345);
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
      final container = _makeContainer(api);
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
      container.read(selectedShipTypeIdProvider.notifier).state = 648;

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
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      container.read(selectedShipTypeIdProvider.notifier).state = 649;

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
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      // shipTypeId set, but region remains null.
      container.read(selectedShipTypeIdProvider.notifier).state = 649;

      await container.read(routesProvider.notifier).calculate();

      expect(apiCalled, isFalse);
      // State should still be null (unchanged from build).
      final result = await container.read(routesProvider.future);
      expect(result, isNull);
    });

    test('calculate() is a no-op when shipTypeId is null', () async {
      bool apiCalled = false;
      final api = FakeTradingApi(
        regionsCallback: () async =>
            RegionsResponse(regions: _fakeRegions, count: 2),
        calculateCallback: (_) async {
          apiCalled = true;
          return _fakeResponse();
        },
      );
      final container = _makeContainer(api);
      addTearDown(container.dispose);

      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      // shipTypeId remains null.

      await container.read(routesProvider.notifier).calculate();

      expect(apiCalled, isFalse);
      final result = await container.read(routesProvider.future);
      expect(result, isNull);
    });
  });

  // ── filtersProvider ────────────────────────────────────────────────────────

  group('filtersProvider', () {
    test('has correct defaults: highSec=true, lowSec=false, nullSec=false',
        () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final filters = container.read(filtersProvider);

      expect(filters.highSec, isTrue);
      expect(filters.lowSec, isFalse);
      expect(filters.nullSec, isFalse);
      expect(filters.minSpread, isNull);
      expect(filters.minProfit, isNull);
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
      final container = _makeContainer(api);
      container.read(selectedRegionProvider.notifier).state = _fakeRegions[0];
      container.read(selectedShipTypeIdProvider.notifier).state = 648;
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

    test('minSpread threshold filters low-spread routes client-side',
        () async {
      final lowSpread = _fakeRoute(typeId: 4, name: 'LowSpread', spreadPercent: 2);
      final highSpread =
          _fakeRoute(typeId: 5, name: 'HighSpread', spreadPercent: 20);
      final container = await containerWithRoutes([lowSpread, highSpread]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(minSpread: 5),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HighSpread']);
    });

    test('minProfit threshold filters low-profit routes client-side',
        () async {
      final lowProfit = _fakeRoute(typeId: 6, name: 'LowProfit', netProfit: 1000);
      final highProfit =
          _fakeRoute(typeId: 7, name: 'HighProfit', netProfit: 500000);
      final container = await containerWithRoutes([lowProfit, highProfit]);
      addTearDown(container.dispose);

      container.read(filtersProvider.notifier).update(
            (f) => f.copyWith(minProfit: 100000),
          );

      final filtered = container.read(filteredRoutesProvider);
      expect(filtered.map((r) => r.itemName), ['HighProfit']);
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

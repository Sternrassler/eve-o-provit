/// Shared fake API implementations for E2E / integration-style widget tests.
///
/// These fakes inject canned responses via Riverpod provider overrides so the
/// full widget tree (auth-gated routing → trading/character screens) can be
/// exercised without a real backend or platform plugins.
library;

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:eve_o_provit/api/hub_comparison_models.dart';
import 'package:eve_o_provit/api/portfolio_models.dart';
import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/auth/auth_controller.dart';
import 'package:eve_o_provit/auth/auth_providers.dart';
import 'package:eve_o_provit/auth/character.dart';
import 'package:eve_o_provit/auth/token_store.dart';
import 'package:eve_o_provit/features/character/character_api.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart'
    show characterApiProvider;
import 'package:eve_o_provit/features/trading/hub_comparison_providers.dart';
import 'package:eve_o_provit/features/trading/providers.dart';
import 'package:eve_o_provit/features/trading/roi_providers.dart';

// ---------------------------------------------------------------------------
// Canned data — shared by both TradingApi and CharacterApi fakes
// ---------------------------------------------------------------------------

/// Two high-sec routes + one low-sec route (hidden by default filter).
TradingRoute fakeRoute({
  int typeId = 34,
  String name = 'Tritanium',
  double minRouteSecurityStatus = 0.9,
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

/// Canned [RouteCalculationResponse] with 2 high-sec + 1 low-sec route.
RouteCalculationResponse fakeRouteResponse() => RouteCalculationResponse(
      regionId: 10000002,
      regionName: 'The Forge',
      shipTypeId: 648,
      shipName: 'Badger',
      cargoCapacity: 50000.0,
      calculationTimeMs: 10,
      routes: [
        fakeRoute(typeId: 34, name: 'Tritanium', minRouteSecurityStatus: 0.9),
        fakeRoute(typeId: 35, name: 'Pyerite', minRouteSecurityStatus: 0.8),
        // low-sec — hidden by default, visible when "Lo" filter is toggled
        fakeRoute(typeId: 36, name: 'Megacyte', minRouteSecurityStatus: 0.3),
      ],
    );

/// Empty [RouteCalculationResponse] (no routes found).
RouteCalculationResponse fakeEmptyRouteResponse() => RouteCalculationResponse(
      regionId: 10000002,
      regionName: 'The Forge',
      shipTypeId: 648,
      shipName: 'Badger',
      cargoCapacity: 50000.0,
      calculationTimeMs: 5,
      routes: const [],
    );

/// Canned [HubComparisonResponse] — Dodixie recommended, Rens has no data.
HubComparisonResponse fakeHubResponse() => const HubComparisonResponse(
      typeId: 34,
      itemName: 'Tritanium',
      bestHubRegionId: 10000032,
      skillsApplied: SkillsApplied(
        applied: true,
        accounting: 5,
        brokerRelations: 5,
        salesTaxRate: 0.025,
        brokerFeeRate: 0.015,
      ),
      hubs: [
        HubRow(
          regionId: 10000032,
          regionName: 'Sinq Laison',
          hubName: 'Dodixie',
          systemId: 30002659,
          tier: 'secondary',
          hasData: true,
          buyPrice: 100,
          sellPrice: 110,
          spreadPercent: 10,
          netMarginPercent: 3.4,
          netProfitPerUnit: 4.1,
          dailyVolume: 125000,
          liquidityScore: 72,
          competition:
              CompetitionInfo(changesPerHour: 42.5, source: 'baseline'),
        ),
        HubRow(
          regionId: 10000002,
          regionName: 'The Forge',
          hubName: 'Jita',
          systemId: 30000142,
          tier: 'primary',
          hasData: true,
          buyPrice: 98,
          sellPrice: 112,
          spreadPercent: 14.3,
          netMarginPercent: 2.1,
          netProfitPerUnit: 2.4,
          dailyVolume: 9000000,
          liquidityScore: 95,
          competition: CompetitionInfo(changesPerHour: 200, source: 'live'),
        ),
        HubRow(
          regionId: 10000030,
          regionName: 'Heimatar',
          hubName: 'Rens',
          systemId: 30002510,
          tier: 'tertiary',
          hasData: false,
          buyPrice: 0,
          sellPrice: 0,
          spreadPercent: 0,
          netMarginPercent: 0,
          netProfitPerUnit: 0,
          dailyVolume: 0,
          liquidityScore: 0,
        ),
      ],
    );

/// Canned [PortfolioResult] — a two-item allocation with totals.
PortfolioResult fakePortfolioResponse() => const PortfolioResult(
      items: [
        PortfolioItem(
          typeId: 34,
          name: 'Tritanium',
          capitalUsed: 1.5e8,
          units: 120000,
          tripsPerDay: 4,
          dailyProfit: 2.5e7,
          roiPercent: 16.7,
        ),
        PortfolioItem(
          typeId: 35,
          name: 'Pyerite',
          capitalUsed: 1.2e8,
          units: 60000,
          tripsPerDay: 3,
          dailyProfit: 1.8e7,
          roiPercent: 15.0,
        ),
      ],
      totalCapitalUsed: 4.8e8,
      totalDailyProfit: 6.1e7,
      timeUsedMin: 118,
      diversificationScore: 72,
      skillsApplied: SkillsApplied(
        applied: true,
        accounting: 5,
        brokerRelations: 5,
        salesTaxRate: 0.025,
        brokerFeeRate: 0.015,
      ),
    );

/// Canned character used in the authenticated auth state.
const fakeCharacter = Character(
  id: 12345678,
  name: 'Test Pilot',
  portraitUrl: '',
  scopes: ['esi-skills.read_skills.v1'],
);

/// Canned [CharacterInfo] returned by the fake CharacterApi.
const fakeCharacterInfo = CharacterInfo(
  characterId: 12345678,
  characterName: 'Test Pilot',
  portraitUrl: '',
  scopes: ['esi-skills.read_skills.v1'],
);

/// Canned [CharacterShip] returned by the fake CharacterApi.
const fakeCharacterShip = CharacterShip(
  shipTypeId: 648,
  shipName: 'My Badger',
  shipItemId: 1000000000002,
  shipTypeName: 'Badger',
  cargoCapacity: 9656.9,
);

/// Canned [CharacterSkillsData] returned by the fake CharacterApi.
final fakeCharacterSkills = CharacterSkillsData(
  characterId: 12345678,
  skills: const {
    'SpaceshipCommand': 5,
    'Navigation': 5,
    'EvasiveManeuvering': 4,
  },
);

// ---------------------------------------------------------------------------
// FakeTradingApi
// ---------------------------------------------------------------------------

/// A [TradingApi] that returns canned data without network access.
class FakeTradingApi extends TradingApi {
  FakeTradingApi() : super(Dio());

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
      fakeRouteResponse();

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
  Future<void> refreshMarket(int regionId, int typeId) async {}

  @override
  Future<ItemSearchResponse> searchItems(String q, {int limit = 20}) async =>
      const ItemSearchResponse(
        items: [
          ItemSearchResult(
            typeId: 34,
            name: 'Tritanium',
            groupName: 'Mineral',
          ),
        ],
      );

  @override
  Future<HubComparisonResponse> compareHubs(int typeId) async =>
      fakeHubResponse();

  @override
  Future<PortfolioResult> optimizePortfolio(PortfolioRequest request) async =>
      fakePortfolioResponse();

  @override
  Future<void> setWaypoint({
    required int destinationId,
    bool addToBeginning = false,
    bool clearOtherWaypoints = false,
  }) async {}
}

// ---------------------------------------------------------------------------
// FakeCharacterApi
// ---------------------------------------------------------------------------

/// Canned [CharacterFitting] returned by the fake CharacterApi.
final fakeCharacterFitting = CharacterFitting(
  characterId: 12345678,
  shipTypeId: 648,
  effectiveCargoM3: 9656.9,
  warpSpeedAuS: 6.87,
  alignTimeSeconds: 4.82,
  baseCargoHoldM3: 2700.0,
  baseWarpSpeedAuS: 3.0,
  fittedModules: 3,
  bonuses: FittingBonuses(
    cargoBonusM3: 6956.9,
    warpSpeedMultiplier: 2.29,
    inertiaModifier: 0.8,
    skillsBonusM3: 2700.0,
    skillsBonusPct: 100.0,
    modulesBonusM3: 4256.9,
  ),
  cached: true,
);

/// A [CharacterApi] that returns canned data without network access.
class FakeCharacterApi extends CharacterApi {
  FakeCharacterApi() : super(Dio());

  @override
  Future<CharacterInfo> me() async => fakeCharacterInfo;

  @override
  Future<CharacterShip> activeShip() async => fakeCharacterShip;

  @override
  Future<CharacterSkillsData> skills(int characterId) async =>
      fakeCharacterSkills;

  @override
  Future<CharacterShipsResponse> ships() async =>
      const CharacterShipsResponse(ships: [], count: 0);

  @override
  Future<CharacterFitting> fitting(int characterId, int shipTypeId) async =>
      fakeCharacterFitting;
}

// ---------------------------------------------------------------------------
// Fake RoutesNotifier — starts with canned data (no calculate() needed)
// ---------------------------------------------------------------------------

/// A [RoutesNotifier] that initialises with [fakeRouteResponse] so the
/// route list is immediately visible without calling calculate().
class FakeRoutesNotifier extends RoutesNotifier {
  @override
  Future<RouteCalculationResponse?> build() async => fakeRouteResponse();

  @override
  Future<void> calculate() async {
    state = AsyncData(fakeRouteResponse());
  }
}

// ---------------------------------------------------------------------------
// Fake HubComparisonNotifier — starts with canned data
// ---------------------------------------------------------------------------

/// A [HubComparisonNotifier] that initialises with [fakeHubResponse] so the
/// comparison table is immediately visible without calling search().
class FakeHubComparisonNotifier extends HubComparisonNotifier {
  @override
  Future<HubComparisonResponse?> build() async => fakeHubResponse();

  @override
  Future<void> search(int typeId) async {
    state = AsyncData(fakeHubResponse());
  }
}

// ---------------------------------------------------------------------------
// Fake RoiOptimizeNotifier — starts with canned data
// ---------------------------------------------------------------------------

/// A [RoiOptimizeNotifier] that initialises with [fakePortfolioResponse] so the
/// allocation table is immediately visible without calling optimize().
class FakeRoiOptimizeNotifier extends RoiOptimizeNotifier {
  @override
  Future<PortfolioResult?> build() async => fakePortfolioResponse();

  @override
  Future<void> optimize(PortfolioRequest request) async {
    state = AsyncData(fakePortfolioResponse());
  }
}

// ---------------------------------------------------------------------------
// Stub AuthControllers
// ---------------------------------------------------------------------------

/// Always returns [Authenticated] with [fakeCharacter].
class FakeAuthControllerAuthenticated extends AuthController {
  @override
  Future<AuthState> build() async =>
      const Authenticated(character: fakeCharacter);

  /// Mirrors the real controller's end state without touching the
  /// [AuthRepository] / [TokenStore] (whose `clear()` calls the
  /// flutter_secure_storage platform plugin, unavailable under `flutter test`).
  @override
  Future<void> logout() async {
    state = const AsyncData(Unauthenticated());
  }
}

/// Always returns [Unauthenticated].
class FakeAuthControllerUnauthenticated extends AuthController {
  @override
  Future<AuthState> build() async => const Unauthenticated();
}

// ---------------------------------------------------------------------------
// Convenience: base provider overrides for authenticated full-app tests
// ---------------------------------------------------------------------------

/// Provider overrides that stub the full auth + API layer for an
/// authenticated session.  Pass to [ProviderScope.overrides].
///
/// Returns a plain [List] whose elements are the opaque override objects
/// produced by Riverpod's `overrideWith` / `overrideWithValue` helpers.
/// The [ProviderScope] constructor accepts `List<dynamic>` at runtime.
List<dynamic> authenticatedOverrides() => [
      // Prevent flutter_secure_storage from being touched.
      tokenStoreProvider.overrideWithValue(TokenStore()),
      authControllerProvider.overrideWith(FakeAuthControllerAuthenticated.new),
      tradingApiProvider.overrideWithValue(FakeTradingApi()),
      routesProvider.overrideWith(FakeRoutesNotifier.new),
      hubComparisonProvider.overrideWith(FakeHubComparisonNotifier.new),
      roiOptimizeProvider.overrideWith(FakeRoiOptimizeNotifier.new),
      // Provide fake character API so ship selector / fitting card work offline.
      characterApiProvider.overrideWithValue(FakeCharacterApi()),
    ];

/// Provider overrides for an unauthenticated session.
List<dynamic> unauthenticatedOverrides() => [
      tokenStoreProvider.overrideWithValue(TokenStore()),
      authControllerProvider
          .overrideWith(FakeAuthControllerUnauthenticated.new),
    ];

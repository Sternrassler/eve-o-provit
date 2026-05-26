/// Riverpod providers for the trading feature.
///
/// Provider graph (no cycles):
///   dioProvider → tradingApiProvider → regionsProvider
///                                    → routesProvider (AsyncNotifier)
///
/// Selection state (independent leaves):
///   selectedRegionProvider, selectedShipTypeIdProvider, selectedRouteProvider
///
/// Filter state:
///   filtersProvider (Notifier)
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/dio_client.dart';
import '../../api/trading_api.dart';
import '../../api/trading_models.dart';

// ---------------------------------------------------------------------------
// TradingApi provider
// ---------------------------------------------------------------------------

/// Provides a [TradingApi] backed by the interceptor-equipped [Dio].
final tradingApiProvider = Provider<TradingApi>((ref) {
  return TradingApi(ref.watch(dioProvider));
});

// ---------------------------------------------------------------------------
// Regions
// ---------------------------------------------------------------------------

/// Fetches all available EVE Online regions from the backend.
/// Cached for the lifetime of the container.
final regionsProvider = FutureProvider<List<Region>>((ref) async {
  final api = ref.watch(tradingApiProvider);
  final response = await api.regions();
  return response.regions;
});

// ---------------------------------------------------------------------------
// Selection state — simple Notifier providers
// ---------------------------------------------------------------------------

/// Notifier for the currently selected [Region]; null until a selection is made.
class SelectedRegionNotifier extends Notifier<Region?> {
  @override
  Region? build() => null;

  /// Sets the selected region.
  void select(Region? region) => state = region;
}

/// Provider for the currently selected [Region].
final selectedRegionProvider =
    NotifierProvider<SelectedRegionNotifier, Region?>(
  SelectedRegionNotifier.new,
);

/// Notifier for the currently selected ship type ID; null until set.
/// (Ship selection UI is implemented in a later task.)
class SelectedShipTypeIdNotifier extends Notifier<int?> {
  @override
  int? build() => null;

  /// Sets the selected ship type ID.
  void select(int? typeId) => state = typeId;
}

/// Provider for the currently selected ship type ID (int?).
final selectedShipTypeIdProvider =
    NotifierProvider<SelectedShipTypeIdNotifier, int?>(
  SelectedShipTypeIdNotifier.new,
);

/// Notifier for the [TradingRoute] displayed in the detail pane.
/// Set when the user taps a route card; null when no card is selected.
class SelectedRouteNotifier extends Notifier<TradingRoute?> {
  @override
  TradingRoute? build() => null;

  /// Sets the selected route.
  void select(TradingRoute? route) => state = route;
}

/// Provider for the currently selected [TradingRoute].
final selectedRouteProvider =
    NotifierProvider<SelectedRouteNotifier, TradingRoute?>(
  SelectedRouteNotifier.new,
);

// ---------------------------------------------------------------------------
// TradingFilters — immutable value class with copyWith
// ---------------------------------------------------------------------------

/// Immutable filter parameters controlling which routes are surfaced.
///
/// Security zones default to high-sec only; min spread / min profit default
/// to null (no threshold applied).
class TradingFilters {
  const TradingFilters({
    this.highSec = true,
    this.lowSec = false,
    this.nullSec = false,
    this.minSpread,
    this.minProfit,
  });

  /// Include high-security systems (security status ≥ 0.5).
  final bool highSec;

  /// Include low-security systems (security status 0.0 – 0.4).
  final bool lowSec;

  /// Include null-security systems (security status < 0.0).
  final bool nullSec;

  /// Minimum spread percentage required for a route (null = no limit).
  final double? minSpread;

  /// Minimum ISK profit required for a route (null = no limit).
  final double? minProfit;

  TradingFilters copyWith({
    bool? highSec,
    bool? lowSec,
    bool? nullSec,
    double? minSpread,
    double? minProfit,
  }) {
    return TradingFilters(
      highSec: highSec ?? this.highSec,
      lowSec: lowSec ?? this.lowSec,
      nullSec: nullSec ?? this.nullSec,
      minSpread: minSpread ?? this.minSpread,
      minProfit: minProfit ?? this.minProfit,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TradingFilters &&
          highSec == other.highSec &&
          lowSec == other.lowSec &&
          nullSec == other.nullSec &&
          minSpread == other.minSpread &&
          minProfit == other.minProfit;

  @override
  int get hashCode =>
      Object.hash(highSec, lowSec, nullSec, minSpread, minProfit);

  @override
  String toString() => 'TradingFilters('
      'highSec: $highSec, lowSec: $lowSec, nullSec: $nullSec, '
      'minSpread: $minSpread, minProfit: $minProfit)';
}

// ---------------------------------------------------------------------------
// FiltersNotifier
// ---------------------------------------------------------------------------

/// Notifier holding [TradingFilters]; exposes a `update` helper.
class FiltersNotifier extends Notifier<TradingFilters> {
  @override
  TradingFilters build() => const TradingFilters();

  /// Replaces the current filter state via [copyWith].
  void update(TradingFilters Function(TradingFilters current) updater) {
    state = updater(state);
  }
}

/// Provider for [TradingFilters].
final filtersProvider =
    NotifierProvider<FiltersNotifier, TradingFilters>(FiltersNotifier.new);

// ---------------------------------------------------------------------------
// RoutesNotifier — AsyncNotifier<RouteCalculationResponse?>
// ---------------------------------------------------------------------------

/// Notifier that owns the route-calculation lifecycle.
///
/// [build] returns null — no calculation has been performed yet.
/// Call [calculate] to trigger a route calculation.  An empty [routes] list
/// is a valid, successful result (the backend may return zero routes when no
/// profitable opportunities exist in the selected region/ship combination);
/// it is NOT treated as an error.  The optional [warning] on the response
/// surfaces informational messages from the backend.
class RoutesNotifier extends AsyncNotifier<RouteCalculationResponse?> {
  @override
  Future<RouteCalculationResponse?> build() async => null;

  /// Runs a route calculation with the current selection.
  ///
  /// Requires [selectedRegionProvider] and [selectedShipTypeIdProvider] to be
  /// non-null; if either is missing, this is a no-op.
  ///
  /// The backend `RouteCalculationRequest` accepts only region/ship (plus
  /// optional fitting/volume params) — it has NO security-zone or
  /// profit/spread filter fields. Like the Next.js web client, filtering by
  /// security zone / min-spread / min-profit happens CLIENT-SIDE on the
  /// returned routes (see [filteredRoutesProvider]). We therefore send only
  /// region_id + ship_type_id, matching the web request body exactly.
  Future<void> calculate() async {
    final region = ref.read(selectedRegionProvider);
    final shipTypeId = ref.read(selectedShipTypeIdProvider);

    if (region == null || shipTypeId == null) return;

    state = const AsyncLoading();

    final request = RouteCalculationRequest(
      regionId: region.id,
      shipTypeId: shipTypeId,
    );

    state = await AsyncValue.guard(() async {
      final api = ref.read(tradingApiProvider);
      return api.calculateRoutes(request);
    });
  }
}

/// Provider for [RoutesNotifier].
final routesProvider =
    AsyncNotifierProvider<RoutesNotifier, RouteCalculationResponse?>(
  RoutesNotifier.new,
);

// ---------------------------------------------------------------------------
// Client-side filtering
// ---------------------------------------------------------------------------

/// Returns the minimum security status along [route] used to classify its
/// security zone — mirrors the Next.js web client:
///   `min_route_security_status ?? min(buy_security_status, sell_security_status)`
///
/// On the Flutter [TradingRoute] all three fields are non-nullable, so we use
/// [TradingRoute.minRouteSecurityStatus] directly (it already represents the
/// most dangerous hop along the route).
double routeMinSecurityStatus(TradingRoute route) {
  return route.minRouteSecurityStatus;
}

/// Returns true when [route] passes the supplied [filters] — applied
/// CLIENT-SIDE, exactly like the Next.js web client.
///
/// Security-zone classification (matches the web):
///   high-sec: sec >= 0.5
///   low-sec : 0.0 < sec < 0.5
///   null-sec: sec <= 0.0
///
/// A route in a zone whose toggle is OFF is hidden. Optional [minSpread] /
/// [minProfit] thresholds (null = no threshold) are also applied client-side
/// against the route's spread percent and net profit.
bool routeMatchesFilters(TradingRoute route, TradingFilters filters) {
  final sec = routeMinSecurityStatus(route);

  final isHighSec = sec >= 0.5;
  final isLowSec = sec > 0.0 && sec < 0.5;
  final isNullSec = sec <= 0.0;

  if (isHighSec && !filters.highSec) return false;
  if (isLowSec && !filters.lowSec) return false;
  if (isNullSec && !filters.nullSec) return false;

  final minSpread = filters.minSpread;
  if (minSpread != null && route.spreadPercent < minSpread) return false;

  final minProfit = filters.minProfit;
  if (minProfit != null && route.netProfit < minProfit) return false;

  return true;
}

/// The routes from [routesProvider], filtered CLIENT-SIDE by [filtersProvider].
///
/// Returns an empty list while no calculation has run yet (data is null) or
/// while loading/in error — those states are surfaced separately by watching
/// [routesProvider] directly. This provider only exposes the *displayable*
/// (filtered) routes so the UI does not re-implement the filter predicate.
final filteredRoutesProvider = Provider<List<TradingRoute>>((ref) {
  final routesAsync = ref.watch(routesProvider);
  final filters = ref.watch(filtersProvider);

  // `.value` is non-null only in the AsyncData state; null while loading,
  // in error, or before the first calculation.
  final response = routesAsync.value;
  if (response == null) return const [];

  return response.routes
      .where((route) => routeMatchesFilters(route, filters))
      .toList();
});

/// Thin API client for the trading/market endpoints.
///
/// All methods take the interceptor-equipped [Dio] (bearer token + 401 retry).
/// Business logic lives in the Riverpod layer above; this class only maps
/// HTTP ↔ DTOs.
library;

import 'package:dio/dio.dart';

import 'asset_models.dart';
import 'hauling_models.dart';
import 'hub_comparison_models.dart';
import 'portfolio_models.dart';
import 'trading_models.dart';

class TradingApi {
  TradingApi(this._dio);

  final Dio _dio;

  // ── GET /api/v1/items/search?q=&limit= ─────────────────────────────────────

  /// Searches EVE items by name. [q] must be at least 3 characters; callers
  /// are responsible for that gate. [limit] caps the number of hits (≤20).
  Future<ItemSearchResponse> searchItems(String q, {int limit = 20}) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/items/search',
      queryParameters: {'q': q, 'limit': limit},
    );
    return ItemSearchResponse.fromJson(response.data!);
  }

  // ── POST /api/v1/trading/hubs/compare ──────────────────────────────────────

  /// Compares an item's station-trading profitability across the major hubs.
  /// Returns a [HubComparisonResponse] with pre-sorted hub rows.
  Future<HubComparisonResponse> compareHubs(int typeId) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/trading/hubs/compare',
      data: {'type_id': typeId},
    );
    return HubComparisonResponse.fromJson(response.data!);
  }

  // ── POST /api/v1/trading/portfolio/optimize ───────────────────────────────

  /// Optimizes a capital allocation across items to maximize daily profit.
  /// Returns a [PortfolioResult] with pre-sorted allocation rows; an empty
  /// `items` list means no viable portfolio for the given budget.
  Future<PortfolioResult> optimizePortfolio(PortfolioRequest request) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/trading/portfolio/optimize',
      data: request.toJson(),
    );
    return PortfolioResult.fromJson(response.data!);
  }

  // ── POST /api/v1/trading/hauling/routes ────────────────────────────────────

  /// Finds profitable station→station hauling routes in the character's
  /// current region plus adjacent regions. Returns a [HaulingResponse] with
  /// routes pre-sorted by isk_per_hour desc; an empty `routes` list means no
  /// profitable routes were found in the neighborhood.
  Future<HaulingResponse> findHaulingRoutes(HaulingRequest request) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/trading/hauling/routes',
      data: request.toJson(),
    );
    return HaulingResponse.fromJson(response.data!);
  }

  // ── GET /api/v1/trading/assets ─────────────────────────────────────────────

  /// Lists the marketable (and non-marketable) items the character owns, used
  /// as the source for the Sell-from-Assets picker.
  Future<AssetsResponse> listAssets() async {
    final response =
        await _dio.get<Map<String, dynamic>>('/api/v1/trading/assets');
    return AssetsResponse.fromJson(response.data!);
  }

  // ── POST /api/v1/trading/assets/sell-options ───────────────────────────────

  /// Finds ranked sell locations (taker model: instant sell into a buy order,
  /// net of sales tax) for an owned asset across the major hubs plus every
  /// station in the item's current region. Returns a [SellOptionsResponse] with
  /// options pre-sorted by total_net desc; an empty `options` list means no
  /// sell locations were found.
  Future<SellOptionsResponse> findSellOptions(
    SellOptionsRequest request,
  ) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/trading/assets/sell-options',
      data: request.toJson(),
    );
    return SellOptionsResponse.fromJson(response.data!);
  }

  // ── POST /api/v1/trading/routes/calculate ──────────────────────────────────

  /// Calculate trading routes for [request].
  /// Returns a [RouteCalculationResponse] with the list of profitable routes.
  Future<RouteCalculationResponse> calculateRoutes(
    RouteCalculationRequest request,
  ) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/trading/routes/calculate',
      data: request.toJson(),
    );
    return RouteCalculationResponse.fromJson(response.data!);
  }

  // ── GET /api/v1/sde/regions ────────────────────────────────────────────────

  /// Returns all available EVE Online regions.
  Future<RegionsResponse> regions() async {
    final response =
        await _dio.get<Map<String, dynamic>>('/api/v1/sde/regions');
    return RegionsResponse.fromJson(response.data!);
  }

  // ── GET /api/v1/market/staleness/:region ──────────────────────────────────

  /// Returns market data staleness info for [regionId].
  Future<MarketDataStalenessResponse> staleness(int regionId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/market/staleness/$regionId',
    );
    return MarketDataStalenessResponse.fromJson(response.data!);
  }

  // ── GET /api/v1/market/:region/:type?refresh=true ─────────────────────────

  /// Triggers a market data refresh for [regionId] / [typeId].
  /// The backend responds with the refreshed market data (204 or 200 body);
  /// we discard the body — callers only need the side-effect.
  Future<void> refreshMarket(int regionId, int typeId) async {
    await _dio.get<void>(
      '/api/v1/market/$regionId/$typeId',
      queryParameters: {'refresh': 'true'},
    );
  }

  // ── POST /api/v1/esi/ui/autopilot/waypoint ────────────────────────────────

  /// Sets an autopilot waypoint in the EVE client.
  ///
  /// [destinationId] — solar system / station / structure ID.
  /// [addToBeginning] — insert at the start of the route (default false).
  /// [clearOtherWaypoints] — clear existing route first (default false).
  Future<void> setWaypoint({
    required int destinationId,
    bool addToBeginning = false,
    bool clearOtherWaypoints = false,
  }) async {
    await _dio.post<void>(
      '/api/v1/esi/ui/autopilot/waypoint',
      data: {
        'destination_id': destinationId,
        'add_to_beginning': addToBeginning,
        'clear_other_waypoints': clearOtherWaypoints,
      },
    );
  }
}

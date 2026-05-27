/// Thin API client for the trading/market endpoints.
///
/// All methods take the interceptor-equipped [Dio] (bearer token + 401 retry).
/// Business logic lives in the Riverpod layer above; this class only maps
/// HTTP ↔ DTOs.
library;

import 'package:dio/dio.dart';

import 'trading_models.dart';

class TradingApi {
  TradingApi(this._dio);

  final Dio _dio;

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

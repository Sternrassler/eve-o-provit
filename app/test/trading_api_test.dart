import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/api/trading_models.dart';

// ---------------------------------------------------------------------------
// Test infrastructure — same _CallbackAdapter pattern as dio_interceptor_test
// ---------------------------------------------------------------------------

/// Minimal [HttpClientAdapter] driven by a callback; avoids the
/// http_mock_adapter "last-match-wins" limitation and keeps tests self-contained.
class _CallbackAdapter implements HttpClientAdapter {
  _CallbackAdapter(this._handler);

  final ResponseBody Function(RequestOptions options) _handler;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async =>
      _handler(options);

  @override
  void close({bool force = false}) {}
}

/// Returns a [ResponseBody] with JSON content-type from [data].
ResponseBody _jsonResponse(Object data, {int statusCode = 200}) {
  return ResponseBody.fromString(
    jsonEncode(data),
    statusCode,
    headers: {
      Headers.contentTypeHeader: ['application/json'],
    },
  );
}

// ---------------------------------------------------------------------------
// Shared fixture — realistic RouteCalculationResponse payload
// ---------------------------------------------------------------------------

const Map<String, dynamic> _routeResponsePayload = {
  'region_id': 10000002,
  'region_name': 'The Forge',
  'ship_type_id': 649,
  'ship_name': 'Badger',
  'cargo_capacity': 50000.0,
  'calculation_time_ms': 312,
  'routes': [
    {
      'item_type_id': 34,
      'item_name': 'Tritanium',
      'buy_system_id': 30000142,
      'buy_system_name': 'Jita',
      'buy_station_id': 60003760,
      'buy_station_name': 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
      'buy_price': 5.50,
      'sell_system_id': 30002187,
      'sell_system_name': 'Amarr',
      'sell_station_id': 60008494,
      'sell_station_name': 'Amarr VIII (Oris) - Emperor Family Academy',
      'sell_price': 6.80,
      'buy_security_status': 1.0,
      'sell_security_status': 1.0,
      'min_route_security_status': 0.5,
      'quantity': 100000,
      'profit_per_unit': 1.3,
      'total_profit': 130000.0,
      'spread_percent': 23.6,
      'travel_time_seconds': 1200.0,
      'round_trip_seconds': 2400.0,
      'isk_per_hour': 195000000.0,
      'jumps': 9,
      'item_volume': 0.01,
      'number_of_tours': 1,
      'profit_per_tour': 130000.0,
      'total_time_minutes': 40.0,
      'base_travel_time_seconds': 1350.0,
      'skilled_travel_time_seconds': 1200.0,
      'base_isk_per_hour': 173000000.0,
      'time_improvement_percent': 11.1,
      'buy_broker_fee': 5000.0,
      'sell_broker_fee': 5500.0,
      'broker_fees': 10500.0,
      'sales_tax': 3500.0,
      'estimated_relist_fee': 2750.0,
      'total_fees': 14000.0,
      'gross_profit': 144000.0,
      'gross_margin_percent': 19.1,
      'net_profit': 116000.0,
      'net_profit_percent': 15.4,
      'cargo_used': 1000.0,
      'cargo_capacity': 50000.0,
      'cargo_utilization': 2.0,
      'base_cargo_capacity': 50000.0,
      'skill_bonus_percent': 0.0,
      'fitting_bonus_m3': 0.0,
      'total_investment': 550000.0,
    },
  ],
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  late Dio dio;
  late TradingApi api;

  setUp(() {
    dio = Dio(BaseOptions(baseUrl: 'http://test.local'));
    api = TradingApi(dio);
  });

  // ── calculateRoutes ────────────────────────────────────────────────────────

  group('TradingApi.calculateRoutes', () {
    test('POSTs to /api/v1/trading/routes/calculate with correct body', () async {
      String? capturedPath;
      String? capturedMethod;
      Map<String, dynamic>? capturedBody;

      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedPath = options.path;
        capturedMethod = options.method;
        capturedBody = options.data as Map<String, dynamic>?;
        return _jsonResponse(_routeResponsePayload);
      });

      final req = RouteCalculationRequest(
        regionId: 10000002,
        shipTypeId: 649,
        cargoCapacity: 50000.0,
      );
      await api.calculateRoutes(req);

      expect(capturedMethod, 'POST');
      expect(capturedPath, '/api/v1/trading/routes/calculate');
      expect(capturedBody?['region_id'], 10000002);
      expect(capturedBody?['ship_type_id'], 649);
      expect(capturedBody?['cargo_capacity'], 50000.0);
    });

    test('parses RouteCalculationResponse from response body', () async {
      dio.httpClientAdapter = _CallbackAdapter(
        (_) => _jsonResponse(_routeResponsePayload),
      );

      final resp = await api.calculateRoutes(
        const RouteCalculationRequest(regionId: 10000002, shipTypeId: 649),
      );

      expect(resp.regionName, 'The Forge');
      expect(resp.shipName, 'Badger');
      expect(resp.cargoCapacity, 50000.0);
      expect(resp.calculationTimeMs, 312);
      expect(resp.routes.length, 1);
      expect(resp.routes[0].itemName, 'Tritanium');
      expect(resp.routes[0].iskPerHour, 195000000.0);
    });

    test('minimal request (no optional fields) omits optional keys from body',
        () async {
      Map<String, dynamic>? capturedBody;

      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedBody = options.data as Map<String, dynamic>?;
        return _jsonResponse(_routeResponsePayload);
      });

      await api.calculateRoutes(
        const RouteCalculationRequest(regionId: 10000002, shipTypeId: 649),
      );

      expect(capturedBody?.containsKey('cargo_capacity'), isFalse);
      expect(capturedBody?.containsKey('warp_speed'), isFalse);
      expect(capturedBody?.containsKey('align_time'), isFalse);
    });
  });

  // ── regions ────────────────────────────────────────────────────────────────

  group('TradingApi.regions', () {
    test('GETs /api/v1/sde/regions and parses RegionsResponse', () async {
      String? capturedPath;

      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedPath = options.path;
        return _jsonResponse({
          'regions': [
            {'id': 10000002, 'name': 'The Forge'},
            {'id': 10000043, 'name': 'Domain'},
          ],
          'count': 2,
        });
      });

      final resp = await api.regions();

      expect(capturedPath, '/api/v1/sde/regions');
      expect(resp.count, 2);
      expect(resp.regions[0].name, 'The Forge');
    });
  });

  // ── staleness ─────────────────────────────────────────────────────────────

  group('TradingApi.staleness', () {
    test('GETs /api/v1/market/staleness/:regionId and parses response',
        () async {
      String? capturedPath;

      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedPath = options.path;
        return _jsonResponse({
          'region_id': 10000002,
          'region_name': 'The Forge',
          'last_update': '2025-05-26T10:00:00Z',
          'age_minutes': 15,
          'status': 'fresh',
          'refresh_allowed': true,
        });
      });

      final resp = await api.staleness(10000002);

      expect(capturedPath, '/api/v1/market/staleness/10000002');
      expect(resp.status, 'fresh');
      expect(resp.ageMinutes, 15);
    });
  });

  // ── refreshMarket ─────────────────────────────────────────────────────────

  group('TradingApi.refreshMarket', () {
    test('GETs /api/v1/market/:regionId/:typeId?refresh=true', () async {
      String? capturedPath;
      Map<String, dynamic>? capturedQuery;

      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedPath = options.path;
        capturedQuery = options.queryParameters;
        return ResponseBody.fromString('', 204);
      });

      await api.refreshMarket(10000002, 34);

      expect(capturedPath, '/api/v1/market/10000002/34');
      expect(capturedQuery?['refresh'], 'true');
    });
  });

  // ── setWaypoint ───────────────────────────────────────────────────────────

  group('TradingApi.setWaypoint', () {
    test('POSTs to /api/v1/esi/ui/autopilot/waypoint with correct body',
        () async {
      String? capturedPath;
      Map<String, dynamic>? capturedBody;

      dio.httpClientAdapter = _CallbackAdapter((options) {
        capturedPath = options.path;
        capturedBody = options.data as Map<String, dynamic>?;
        return ResponseBody.fromString('', 204);
      });

      await api.setWaypoint(
        destinationId: 30000142,
        addToBeginning: true,
        clearOtherWaypoints: false,
      );

      expect(capturedPath, '/api/v1/esi/ui/autopilot/waypoint');
      expect(capturedBody?['destination_id'], 30000142);
      expect(capturedBody?['add_to_beginning'], true);
      expect(capturedBody?['clear_other_waypoints'], false);
    });
  });
}

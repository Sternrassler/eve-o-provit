import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_models.dart';

void main() {
  // ── RouteCalculationResponse.fromJson ──────────────────────────────────────

  group('RouteCalculationResponse.fromJson', () {
    // Realistic sample shaped after the backend RouteCalculationResponse model
    // (json tags from models/trading.go).
    const Map<String, dynamic> sample = {
      'region_id': 10000002,
      'region_name': 'The Forge',
      'ship_type_id': 649,
      'ship_name': 'Badger',
      'cargo_capacity': 50000.0,
      'calculation_time_ms': 432,
      'warning': '',
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
          // volume_metrics omitted (optional)
          'liquidation_days': 2.5,
          'daily_profit': 46400.0,
        },
      ],
    };

    test('maps top-level scalar fields', () {
      final resp = RouteCalculationResponse.fromJson(sample);

      expect(resp.regionId, 10000002);
      expect(resp.regionName, 'The Forge');
      expect(resp.shipTypeId, 649);
      expect(resp.shipName, 'Badger');
      expect(resp.cargoCapacity, 50000.0);
      expect(resp.calculationTimeMs, 432);
    });

    test('maps warning field (empty string)', () {
      final resp = RouteCalculationResponse.fromJson(sample);
      expect(resp.warning, '');
    });

    test('warning is null when absent', () {
      final noWarning = Map<String, dynamic>.from(sample)..remove('warning');
      final resp = RouteCalculationResponse.fromJson(noWarning);
      expect(resp.warning, isNull);
    });

    test('routes list has correct length', () {
      final resp = RouteCalculationResponse.fromJson(sample);
      expect(resp.routes.length, 1);
    });

    group('routes[0] TradingRoute fields', () {
      late TradingRoute route;

      setUp(() {
        route = RouteCalculationResponse.fromJson(sample).routes[0];
      });

      test('item identification', () {
        expect(route.itemTypeId, 34);
        expect(route.itemName, 'Tritanium');
      });

      test('buy-side fields', () {
        expect(route.buySystemId, 30000142);
        expect(route.buySystemName, 'Jita');
        expect(route.buyStationId, 60003760);
        expect(route.buyStationName,
            'Jita IV - Moon 4 - Caldari Navy Assembly Plant');
        expect(route.buyPrice, 5.50);
        expect(route.buySecurityStatus, 1.0);
      });

      test('sell-side fields', () {
        expect(route.sellSystemId, 30002187);
        expect(route.sellSystemName, 'Amarr');
        expect(route.sellStationId, 60008494);
        expect(route.sellStationName,
            'Amarr VIII (Oris) - Emperor Family Academy');
        expect(route.sellPrice, 6.80);
        expect(route.sellSecurityStatus, 1.0);
      });

      test('profit & volume fields', () {
        expect(route.quantity, 100000);
        expect(route.profitPerUnit, 1.3);
        expect(route.totalProfit, 130000.0);
        expect(route.spreadPercent, 23.6);
        expect(route.iskPerHour, 195000000.0);
        expect(route.jumps, 9);
        expect(route.itemVolume, 0.01);
      });

      test('fees fields', () {
        expect(route.buyBrokerFee, 5000.0);
        expect(route.sellBrokerFee, 5500.0);
        expect(route.brokerFees, 10500.0);
        expect(route.salesTax, 3500.0);
        expect(route.totalFees, 14000.0);
        expect(route.netProfit, 116000.0);
        expect(route.netProfitPercent, 15.4);
      });

      test('cargo fields', () {
        expect(route.cargoUsed, 1000.0);
        expect(route.cargoCapacity, 50000.0);
        expect(route.cargoUtilization, 2.0);
        expect(route.totalInvestment, 550000.0);
      });

      test('optional volume metrics is null when absent', () {
        expect(route.volumeMetrics, isNull);
      });

      test('optional liquidation_days is mapped', () {
        expect(route.liquidationDays, 2.5);
      });
    });
  });

  // ── RouteCalculationRequest.toJson ────────────────────────────────────────

  group('RouteCalculationRequest.toJson', () {
    test('minimal request produces correct keys', () {
      final req = RouteCalculationRequest(regionId: 10000002, shipTypeId: 649);
      final json = req.toJson();

      expect(json['region_id'], 10000002);
      expect(json['ship_type_id'], 649);
    });

    test('optional fields are omitted when null', () {
      final req = RouteCalculationRequest(regionId: 10000002, shipTypeId: 649);
      final json = req.toJson();

      expect(json.containsKey('cargo_capacity'), isFalse);
      expect(json.containsKey('warp_speed'), isFalse);
      expect(json.containsKey('align_time'), isFalse);
      expect(json.containsKey('min_daily_volume'), isFalse);
      expect(json.containsKey('max_liquidation_days'), isFalse);
    });

    test('optional fields are included when set', () {
      final req = RouteCalculationRequest(
        regionId: 10000002,
        shipTypeId: 649,
        cargoCapacity: 62500.0,
        warpSpeed: 4.2,
        alignTime: 4.8,
        minDailyVolume: 100.0,
        maxLiquidationDays: 7.0,
        includeVolumeMetrics: true,
      );
      final json = req.toJson();

      expect(json['cargo_capacity'], 62500.0);
      expect(json['warp_speed'], 4.2);
      expect(json['align_time'], 4.8);
      expect(json['min_daily_volume'], 100.0);
      expect(json['max_liquidation_days'], 7.0);
      expect(json['include_volume_metrics'], true);
    });
  });

  // ── Region.fromJson ────────────────────────────────────────────────────────

  group('Region.fromJson', () {
    test('maps id and name', () {
      final region = Region.fromJson({'id': 10000002, 'name': 'The Forge'});
      expect(region.id, 10000002);
      expect(region.name, 'The Forge');
    });
  });

  // ── RegionsResponse.fromJson ───────────────────────────────────────────────

  group('RegionsResponse.fromJson', () {
    test('maps regions list and count', () {
      final resp = RegionsResponse.fromJson({
        'regions': [
          {'id': 10000002, 'name': 'The Forge'},
          {'id': 10000043, 'name': 'Domain'},
        ],
        'count': 2,
      });

      expect(resp.count, 2);
      expect(resp.regions.length, 2);
      expect(resp.regions[0].name, 'The Forge');
      expect(resp.regions[1].id, 10000043);
    });
  });

  // ── MarketDataStalenessResponse.fromJson ───────────────────────────────────

  group('MarketDataStalenessResponse.fromJson', () {
    test('maps all fields', () {
      final resp = MarketDataStalenessResponse.fromJson({
        'region_id': 10000002,
        'region_name': 'The Forge',
        'last_update': '2025-05-26T10:00:00Z',
        'age_minutes': 15,
        'status': 'fresh',
        'refresh_allowed': true,
      });

      expect(resp.regionId, 10000002);
      expect(resp.regionName, 'The Forge');
      expect(resp.ageMinutes, 15);
      expect(resp.status, 'fresh');
      expect(resp.refreshAllowed, isTrue);
      expect(resp.lastUpdate, DateTime.utc(2025, 5, 26, 10, 0, 0));
    });

    test('accepts a float age_minutes (backend returns epoch diff / 60)', () {
      final resp = MarketDataStalenessResponse.fromJson({
        'region_id': 10000002,
        'region_name': 'The Forge',
        'last_update': '2025-05-26T10:00:00Z',
        'age_minutes': 0.05,
        'status': 'fresh',
        'refresh_allowed': true,
      });

      expect(resp.ageMinutes, 0.05);
    });

    test('tolerates null last_update / age_minutes (region without data)', () {
      final resp = MarketDataStalenessResponse.fromJson({
        'region_id': 10000002,
        'region_name': 'The Forge',
        'last_update': null,
        'age_minutes': null,
        'status': 'very_stale',
        'refresh_allowed': true,
      });

      expect(resp.lastUpdate, isNull);
      expect(resp.ageMinutes, isNull);
      expect(resp.status, 'very_stale');
    });
  });

  // ── VolumeMetrics.fromJson ─────────────────────────────────────────────────

  group('VolumeMetrics.fromJson', () {
    test('maps all fields', () {
      final vm = VolumeMetrics.fromJson({
        'type_id': 34,
        'region_id': 10000002,
        'daily_volume_avg': 1500000.0,
        'daily_isk_turnover': 8250000.0,
        'liquidity_score': 85,
        'data_days': 30,
      });

      expect(vm.typeId, 34);
      expect(vm.regionId, 10000002);
      expect(vm.dailyVolumeAvg, 1500000.0);
      expect(vm.dailyIskTurnover, 8250000.0);
      expect(vm.liquidityScore, 85);
      expect(vm.dataDays, 30);
    });
  });
}

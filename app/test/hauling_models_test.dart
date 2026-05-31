/// Unit tests for HaulingResponse.fromJson — exercises the null/float-robust
/// parsing required for the mobile DTO layer.
library;

import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/hauling_models.dart';

void main() {
  group('HaulingRequest.toJson', () {
    test('maps required fields and omits cargo_capacity when null', () {
      const req = HaulingRequest(
        originRegionId: 0,
        shipTypeId: 648,
        capital: 500000000,
        avoidLowSec: true,
        maxRoutes: 15,
      );
      expect(req.toJson(), {
        'origin_region_id': 0,
        'ship_type_id': 648,
        'capital': 500000000.0,
        'avoid_low_sec': true,
        'max_routes': 15,
      });
      expect(req.toJson().containsKey('cargo_capacity'), isFalse);
    });

    test('includes cargo_capacity override when set (current-ship cargo)', () {
      const req = HaulingRequest(
        originRegionId: 0,
        shipTypeId: 648,
        capital: 1,
        avoidLowSec: false,
        maxRoutes: 1,
        cargoCapacity: 9656.9,
      );
      expect(req.toJson()['cargo_capacity'], 9656.9);
    });
  });

  group('HaulingResponse.fromJson', () {
    const Map<String, dynamic> sample = {
      'origin_region_id': 10000002,
      'regions_scanned': [10000002, 10000016],
      'routes': [
        {
          'buy_system_name': 'Osmon',
          'buy_station_name': 'Osmon V - Moon 5 - Some Station',
          'buy_station_id': 60000001,
          'sell_system_name': 'Jita',
          'sell_station_name': 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
          'sell_station_id': 60003760,
          'jumps': 7,
          'travel_time_min': 12.5,
          'security_risk': 'safe',
          'total_profit': 1.2e8,
          'total_capital': 3.4e8,
          'total_volume': 2400,
          'isk_per_hour': 4.8e8,
          'items': [
            {
              'type_id': 34,
              'name': 'Meta-4 Module A',
              'quantity': 50,
              'buy_price': 1.0e6,
              'sell_price': 1.9e6,
              'unit_volume': 10,
              'profit': 4.5e7,
              'profit_per_m3': 90000,
            },
          ],
        },
      ],
      'skills_applied': {
        'applied': true,
        'accounting': 5,
        'broker_relations': 5,
        'sales_tax_rate': 0.025,
        'broker_fee_rate': 0.015,
      },
    };

    test('parses all fields including nested route + item', () {
      final resp = HaulingResponse.fromJson(sample);

      expect(resp.originRegionId, 10000002);
      expect(resp.regionsScanned, [10000002, 10000016]);
      expect(resp.isEmpty, isFalse);
      expect(resp.routes, hasLength(1));
      expect(resp.skillsApplied.applied, isTrue);
      expect(resp.skillsApplied.accounting, 5);

      final route = resp.routes.first;
      expect(route.buySystemName, 'Osmon');
      expect(route.sellSystemName, 'Jita');
      expect(route.buyStationId, 60000001);
      expect(route.sellStationId, 60003760);
      expect(route.jumps, 7);
      expect(route.travelTimeMin, 12.5);
      expect(route.securityRisk, 'safe');
      expect(route.totalProfit, 1.2e8);
      expect(route.totalCapital, 3.4e8);
      expect(route.totalVolume, 2400);
      expect(route.iskPerHour, 4.8e8);

      final item = route.items.single;
      expect(item.typeId, 34);
      expect(item.name, 'Meta-4 Module A');
      expect(item.quantity, 50);
      expect(item.buyPrice, 1.0e6);
      expect(item.sellPrice, 1.9e6);
      expect(item.unitVolume, 10);
      expect(item.profit, 4.5e7);
      expect(item.profitPerM3, 90000);
    });

    test('tolerates missing / null fields (defaults, not exceptions)', () {
      final resp = HaulingResponse.fromJson(const {
        'routes': [
          {
            'buy_system_name': 'A',
            'sell_system_name': 'B',
            // everything numeric absent
          },
        ],
      });

      expect(resp.originRegionId, 0);
      expect(resp.regionsScanned, isEmpty);
      expect(resp.skillsApplied.applied, isFalse);

      final route = resp.routes.single;
      expect(route.jumps, 0);
      expect(route.travelTimeMin, 0);
      expect(route.securityRisk, 'safe'); // defaulted
      expect(route.iskPerHour, 0);
      expect(route.items, isEmpty);
    });

    test('empty routes list is a valid "no routes" result', () {
      final resp = HaulingResponse.fromJson(const {
        'origin_region_id': 10000002,
        'regions_scanned': [10000002],
        'routes': [],
      });
      expect(resp.isEmpty, isTrue);
      expect(resp.routes, isEmpty);
    });
  });
}

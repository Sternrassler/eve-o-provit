/// Unit tests for asset_models — exercises the null/float-robust parsing
/// required for the mobile DTO layer (Sell-from-Assets, issue #107).
library;

import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/asset_models.dart';

void main() {
  group('AssetsResponse.fromJson', () {
    test('parses assets + count', () {
      final resp = AssetsResponse.fromJson(const {
        'assets': [
          {
            'type_id': 34,
            'name': 'Tritanium',
            'quantity': 120000,
            'location_id': 60003760,
            'location_name': 'Jita IV - Moon 4 - CNAP',
            'system_id': 30000142,
            'region_id': 10000002,
            'marketable': true,
          },
          {
            'type_id': 99,
            'name': 'Blueprint Copy',
            'quantity': 1,
            'location_id': 60003760,
            'location_name': 'Jita IV - Moon 4 - CNAP',
            'system_id': 30000142,
            'region_id': 10000002,
            'marketable': false,
          },
        ],
        'count': 2,
      });

      expect(resp.count, 2);
      expect(resp.assets, hasLength(2));
      expect(resp.isEmpty, isFalse);

      final trit = resp.assets.first;
      expect(trit.typeId, 34);
      expect(trit.name, 'Tritanium');
      expect(trit.quantity, 120000);
      expect(trit.locationId, 60003760);
      expect(trit.locationName, 'Jita IV - Moon 4 - CNAP');
      expect(trit.systemId, 30000142);
      expect(trit.regionId, 10000002);
      expect(trit.marketable, isTrue);

      expect(resp.assets[1].marketable, isFalse);
    });

    test('tolerates missing / null fields', () {
      final resp = AssetsResponse.fromJson(const {
        'assets': [
          {'name': 'Mystery'},
        ],
      });
      expect(resp.count, 0);
      final a = resp.assets.single;
      expect(a.typeId, 0);
      expect(a.quantity, 0);
      expect(a.locationId, 0);
      expect(a.marketable, isFalse);
    });

    test('empty assets is a valid result', () {
      final resp = AssetsResponse.fromJson(const {'assets': [], 'count': 0});
      expect(resp.isEmpty, isTrue);
      expect(resp.assets, isEmpty);
    });
  });

  group('SellOptionsResponse.fromJson', () {
    const Map<String, dynamic> sample = {
      'type_id': 34,
      'name': 'Tritanium',
      'quantity': 120000,
      'origin_system_id': 30000142,
      'best': {
        'scope': 'hub',
        'region_id': 10000002,
        'region_name': 'The Forge',
        'station_id': 60003760,
        'station_name': 'Jita IV - Moon 4 - CNAP',
        'system_name': 'Jita',
        'buy_price': 5.5,
        'unit_net': 5.36,
        'total_net': 643200,
        'jumps': 0,
        'travel_time_min': 0,
        'security_risk': 'safe',
        'has_data': true,
      },
      'options': [
        {
          'scope': 'hub',
          'region_id': 10000002,
          'region_name': 'The Forge',
          'station_id': 60003760,
          'station_name': 'Jita IV - Moon 4 - CNAP',
          'system_name': 'Jita',
          'buy_price': 5.5,
          'unit_net': 5.36,
          'total_net': 643200,
          'jumps': 0,
          'travel_time_min': 0,
          'security_risk': 'safe',
          'has_data': true,
        },
        {
          'scope': 'current_region',
          'region_id': 10000002,
          'region_name': 'The Forge',
          'station_id': 60000001,
          'station_name': 'Some Outpost',
          'system_name': 'Sobaseki',
          'security_risk': 'danger',
          'has_data': false,
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

    test('parses all fields including best + nested options', () {
      final resp = SellOptionsResponse.fromJson(sample);

      expect(resp.typeId, 34);
      expect(resp.name, 'Tritanium');
      expect(resp.quantity, 120000);
      expect(resp.originSystemId, 30000142);
      expect(resp.isEmpty, isFalse);
      expect(resp.options, hasLength(2));
      expect(resp.skillsApplied.applied, isTrue);
      expect(resp.skillsApplied.salesTaxRate, 0.025);

      final best = resp.best!;
      expect(best.scope, 'hub');
      expect(best.isHub, isTrue);
      expect(best.regionName, 'The Forge');
      expect(best.stationId, 60003760);
      expect(best.systemName, 'Jita');
      expect(best.buyPrice, 5.5);
      expect(best.unitNet, 5.36);
      expect(best.totalNet, 643200);
      expect(best.jumps, 0);
      expect(best.travelTimeMin, 0);
      expect(best.securityRisk, 'safe');
      expect(best.hasData, isTrue);

      // has_data:false option defaults numeric fields to 0.
      final noData = resp.options[1];
      expect(noData.hasData, isFalse);
      expect(noData.scope, 'current_region');
      expect(noData.isHub, isFalse);
      expect(noData.buyPrice, 0);
      expect(noData.totalNet, 0);
      expect(noData.securityRisk, 'danger');
    });

    test('tolerates missing / null fields (defaults, not exceptions)', () {
      final resp = SellOptionsResponse.fromJson(const {
        'options': [
          {
            'system_name': 'Jita',
            // everything else absent
          },
        ],
      });

      expect(resp.typeId, 0);
      expect(resp.name, '');
      expect(resp.quantity, 0);
      expect(resp.skillsApplied.applied, isFalse);
      expect(resp.best, isNull);

      final opt = resp.options.single;
      expect(opt.scope, 'current_region'); // defaulted
      expect(opt.jumps, 0);
      expect(opt.totalNet, 0);
      expect(opt.securityRisk, 'safe'); // defaulted
      expect(opt.hasData, isFalse);
    });

    test('best:null + empty options is a valid result', () {
      final resp = SellOptionsResponse.fromJson(const {
        'type_id': 34,
        'name': 'Tritanium',
        'quantity': 120000,
        'origin_system_id': 30000142,
        'best': null,
        'options': [],
      });
      expect(resp.best, isNull);
      expect(resp.isEmpty, isTrue);
      expect(resp.options, isEmpty);
    });
  });
}

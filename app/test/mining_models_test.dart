/// Unit tests for OreRankingResponse.fromJson — exercises the null/float-robust
/// parsing required for the mobile DTO layer (mirrors hauling_models_test.dart).
library;

import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/mining_models.dart';

void main() {
  group('OreRankingRequest.toJson', () {
    test('maps region_id and sec_band', () {
      const req = OreRankingRequest(regionId: 0, secBand: 'high');
      expect(req.toJson(), {'region_id': 0, 'sec_band': 'high'});
    });

    test('serializes low and null bands correctly', () {
      expect(
        const OreRankingRequest(regionId: 10000002, secBand: 'low').toJson(),
        {'region_id': 10000002, 'sec_band': 'low'},
      );
      expect(
        const OreRankingRequest(regionId: 10000002, secBand: 'null').toJson(),
        {'region_id': 10000002, 'sec_band': 'null'},
      );
    });
  });

  group('OreRankRow.fromJson', () {
    const Map<String, dynamic> sample = {
      'ore_type_id': 1230,
      'ore_name': 'Veldspar',
      'mining_m3_per_hour': 3600.0,
      'raw_isk_per_hour': 4200000.0,
      'refine_isk_per_hour': 5100000.0,
      'raw_net_per_m3': 1166.67,
      'refine_net_per_m3': 1416.67,
      'best': 'refine',
      'delta_isk_per_hour': 900000.0,
      'best_station_id': 60003760,
      'best_station_tax': 0.05,
    };

    test('parses all fields correctly', () {
      final row = OreRankRow.fromJson(sample);

      expect(row.oreTypeId, 1230);
      expect(row.oreName, 'Veldspar');
      expect(row.miningM3PerHour, 3600.0);
      expect(row.rawIskPerHour, 4200000.0);
      expect(row.refineIskPerHour, 5100000.0);
      expect(row.rawNetPerM3, closeTo(1166.67, 0.01));
      expect(row.refineNetPerM3, closeTo(1416.67, 0.01));
      expect(row.best, 'refine');
      expect(row.deltaIskPerHour, 900000.0);
      expect(row.bestStationId, 60003760);
      expect(row.bestStationTax, 0.05);
    });

    test('tolerates missing / null fields (defaults, not exceptions)', () {
      final row = OreRankRow.fromJson(const {'ore_name': 'Plagioclase'});

      expect(row.oreTypeId, 0);
      expect(row.oreName, 'Plagioclase');
      expect(row.miningM3PerHour, 0);
      expect(row.rawIskPerHour, 0);
      expect(row.refineIskPerHour, 0);
      expect(row.rawNetPerM3, 0);
      expect(row.refineNetPerM3, 0);
      expect(row.best, 'raw'); // defaulted
      expect(row.deltaIskPerHour, 0);
      expect(row.bestStationId, isNull);
      expect(row.bestStationTax, 0);
    });

    test('accepts int values for double fields (num-robust)', () {
      final row = OreRankRow.fromJson(const {
        'ore_type_id': 1230,
        'ore_name': 'Scordite',
        'mining_m3_per_hour': 3000, // int
        'raw_isk_per_hour': 3000000, // int
        'refine_isk_per_hour': 3500000, // int
        'raw_net_per_m3': 1000, // int
        'refine_net_per_m3': 1166, // int
        'best': 'raw',
        'delta_isk_per_hour': 500000, // int
        'best_station_tax': 0, // int zero
      });

      expect(row.miningM3PerHour, 3000.0);
      expect(row.rawIskPerHour, 3000000.0);
      expect(row.refineIskPerHour, 3500000.0);
      expect(row.bestStationTax, 0.0);
    });

    test('best_station_id nullable — absent maps to null', () {
      final row = OreRankRow.fromJson(const {
        'ore_type_id': 1230,
        'ore_name': 'Kernite',
        'best': 'raw',
        // best_station_id deliberately absent
      });
      expect(row.bestStationId, isNull);
    });

    test('parses location fields: reprocess station, raw_sell, materials', () {
      final row = OreRankRow.fromJson(const {
        'ore_type_id': 1230,
        'ore_name': 'Veldspar',
        'best': 'refine',
        'best_station_name': 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
        'best_station_system': 'Jita',
        'raw_sell': {
          'station_name': 'Amarr VIII - Emperor Family Academy',
          'system_name': 'Amarr',
          'is_structure': false,
        },
        'materials': [
          {
            'material_type_id': 34,
            'material_name': 'Tritanium',
            'effective_qty': 12345,
            'buy_price': 5.5,
            'sell': {
              'station_name': 'Dodixie IX - Moon 20 - Federation Navy',
              'system_name': 'Dodixie',
              'is_structure': false,
            },
          },
        ],
      });

      expect(row.bestStationName,
          'Jita IV - Moon 4 - Caldari Navy Assembly Plant');
      expect(row.bestStationSystem, 'Jita');
      expect(row.rawSell, isNotNull);
      expect(row.rawSell!.isStructure, isFalse);
      expect(row.rawSell!.stationName, 'Amarr VIII - Emperor Family Academy');
      expect(row.rawSell!.systemName, 'Amarr');

      expect(row.materials, hasLength(1));
      final mat = row.materials.first;
      expect(mat.materialTypeId, 34);
      expect(mat.materialName, 'Tritanium');
      expect(mat.effectiveQty, 12345);
      expect(mat.buyPrice, 5.5);
      expect(mat.sell.systemName, 'Dodixie');
    });

    test('location fields default when absent (no exceptions)', () {
      final row = OreRankRow.fromJson(const {
        'ore_type_id': 1228,
        'ore_name': 'Scordite',
        'best': 'raw',
      });
      expect(row.bestStationName, isNull);
      expect(row.bestStationSystem, isNull);
      expect(row.rawSell, isNull);
      expect(row.materials, isEmpty);
    });

    test('parses estimate + multiplier fields; defaults to exact', () {
      final est = OreRankRow.fromJson(const {
        'ore_type_id': 1228,
        'ore_name': 'Scordite',
        'best': 'raw',
        'is_estimate': true,
        'estimate_reason': 'Kein Crystal für dieses Erz',
        'hull_yield_multiplier': 1.495,
        'crystal_multiplier': 1.0,
      });
      expect(est.isEstimate, isTrue);
      expect(est.estimateReason, 'Kein Crystal für dieses Erz');
      expect(est.hullYieldMultiplier, closeTo(1.495, 0.001));

      final exact = OreRankRow.fromJson(
          const {'ore_type_id': 1230, 'ore_name': 'Veldspar', 'best': 'refine'});
      expect(exact.isEstimate, isFalse);
      expect(exact.crystalMultiplier, 1.0);
    });

    test('citadel sell location parses is_structure=true', () {
      final row = OreRankRow.fromJson(const {
        'ore_type_id': 1228,
        'ore_name': 'Scordite',
        'best': 'raw',
        'raw_sell': {'is_structure': true},
      });
      expect(row.rawSell, isNotNull);
      expect(row.rawSell!.isStructure, isTrue);
      expect(row.rawSell!.stationName, isNull);
    });

    test('parses effective-ISK/h cycle fields; defaults to zero', () {
      final r = OreRankRow.fromJson(const {
        'ore_type_id': 1230, 'ore_name': 'Veldspar', 'best': 'refine',
        'effective_isk_per_hour': 950000, 'cycle_minutes': 12.5,
        'route_jumps': 3, 'sell_system_name': 'Amarr', 'load_volume_m3': 11500,
      });
      expect(r.effectiveIskPerHour, closeTo(950000, 0.1));
      expect(r.cycleMinutes, closeTo(12.5, 0.01));
      expect(r.routeJumps, 3);
      expect(r.sellSystemName, 'Amarr');

      final d = OreRankRow.fromJson(const {'ore_type_id': 1, 'ore_name': 'X', 'best': 'raw'});
      expect(d.effectiveIskPerHour, 0.0);
      expect(d.routeJumps, 0);
    });
  });

  group('OreRankingResponse.fromJson', () {
    const Map<String, dynamic> sample = {
      'region_id': 10000002,
      'sec_band': 'high',
      'no_mining_setup': false,
      'rows': [
        {
          'ore_type_id': 1230,
          'ore_name': 'Veldspar',
          'mining_m3_per_hour': 3600.0,
          'raw_isk_per_hour': 4200000.0,
          'refine_isk_per_hour': 5100000.0,
          'raw_net_per_m3': 1166.67,
          'refine_net_per_m3': 1416.67,
          'best': 'refine',
          'delta_isk_per_hour': 900000.0,
          'best_station_id': 60003760,
          'best_station_tax': 0.05,
        },
        {
          'ore_type_id': 1228,
          'ore_name': 'Scordite',
          'mining_m3_per_hour': 3000.0,
          'raw_isk_per_hour': 3100000.0,
          'refine_isk_per_hour': 3400000.0,
          'raw_net_per_m3': 1033.33,
          'refine_net_per_m3': 1133.33,
          'best': 'refine',
          'delta_isk_per_hour': 300000.0,
          'best_station_id': null,
          'best_station_tax': 0.04,
        },
      ],
    };

    test('parses all fields including nested rows', () {
      final resp = OreRankingResponse.fromJson(sample);

      expect(resp.regionId, 10000002);
      expect(resp.secBand, 'high');
      expect(resp.noMiningSetup, isFalse);
      expect(resp.isEmpty, isFalse);
      expect(resp.rows, hasLength(2));

      final first = resp.rows.first;
      expect(first.oreName, 'Veldspar');
      expect(first.best, 'refine');
      expect(first.bestStationId, 60003760);

      final second = resp.rows.last;
      expect(second.oreName, 'Scordite');
      expect(second.bestStationId, isNull);
    });

    test('no_mining_setup=true is parsed correctly', () {
      final resp = OreRankingResponse.fromJson(const {
        'region_id': 10000002,
        'sec_band': 'high',
        'no_mining_setup': true,
        'rows': [],
      });

      expect(resp.noMiningSetup, isTrue);
      expect(resp.isEmpty, isTrue);
    });

    test('tolerates missing / null fields (defaults, not exceptions)', () {
      final resp = OreRankingResponse.fromJson(const {});

      expect(resp.regionId, 0);
      expect(resp.secBand, 'high');
      expect(resp.noMiningSetup, isFalse);
      expect(resp.rows, isEmpty);
      expect(resp.isEmpty, isTrue);
    });

    test('empty rows list is a valid "no ores" result', () {
      final resp = OreRankingResponse.fromJson(const {
        'region_id': 10000002,
        'sec_band': 'null',
        'no_mining_setup': false,
        'rows': [],
      });
      expect(resp.isEmpty, isTrue);
      expect(resp.rows, isEmpty);
    });
  });
}

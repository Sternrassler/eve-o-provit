import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/hub_comparison_models.dart';

void main() {
  // ── HubComparisonResponse.fromJson ─────────────────────────────────────────

  group('HubComparisonResponse.fromJson', () {
    // Realistic sample: one full hub, one no-data hub, one hub with missing /
    // null numeric fields — exercises the null/float-robust parsing.
    const Map<String, dynamic> sample = {
      'type_id': 34,
      'item_name': 'Tritanium',
      'best_hub_region_id': 10000032,
      'skills_applied': {
        'applied': true,
        'accounting': 5,
        'broker_relations': 5,
        'sales_tax_rate': 0.025,
        'broker_fee_rate': 0.015,
      },
      'hubs': [
        {
          'region_id': 10000032,
          'region_name': 'Sinq Laison',
          'hub_name': 'Dodixie',
          'system_id': 30002659,
          'tier': 'secondary',
          'has_data': true,
          'buy_price': 100.0,
          'sell_price': 110.0,
          'spread_percent': 10.0,
          'net_margin_percent': 3.4,
          'net_profit_per_unit': 4.1,
          'daily_volume': 125000,
          'liquidity_score': 72,
          'competition': {
            'changes_per_hour': 42.5,
            'source': 'baseline',
          },
        },
        // Hub with missing/null numeric fields (live competition source).
        {
          'region_id': 10000002,
          'region_name': 'The Forge',
          'hub_name': 'Jita',
          'system_id': 30000142,
          'tier': 'primary',
          'has_data': true,
          'buy_price': null,
          // sell_price absent entirely
          'spread_percent': 5,
          'net_margin_percent': null,
          'daily_volume': null,
          'liquidity_score': 90,
          'competition': {
            'changes_per_hour': 120,
            'source': 'live',
          },
        },
        // No-data hub: numeric fields absent, competition absent.
        {
          'region_id': 10000030,
          'region_name': 'Heimatar',
          'hub_name': 'Rens',
          'system_id': 30002510,
          'tier': 'tertiary',
          'has_data': false,
        },
      ],
    };

    test('maps top-level scalar fields', () {
      final resp = HubComparisonResponse.fromJson(sample);
      expect(resp.typeId, 34);
      expect(resp.itemName, 'Tritanium');
      expect(resp.bestHubRegionId, 10000032);
      expect(resp.hubs.length, 3);
    });

    test('maps skills_applied object', () {
      final s = HubComparisonResponse.fromJson(sample).skillsApplied;
      expect(s.applied, isTrue);
      expect(s.accounting, 5);
      expect(s.brokerRelations, 5);
      expect(s.salesTaxRate, 0.025);
      expect(s.brokerFeeRate, 0.015);
    });

    test('skills_applied defaults when object absent', () {
      final noSkills = Map<String, dynamic>.from(sample)
        ..remove('skills_applied');
      final s = HubComparisonResponse.fromJson(noSkills).skillsApplied;
      expect(s.applied, isFalse);
      expect(s.accounting, 0);
      expect(s.salesTaxRate, 0);
    });

    group('full hub row', () {
      late HubRow hub;
      setUp(() => hub = HubComparisonResponse.fromJson(sample).hubs[0]);

      test('identification + tier', () {
        expect(hub.regionId, 10000032);
        expect(hub.regionName, 'Sinq Laison');
        expect(hub.hubName, 'Dodixie');
        expect(hub.systemId, 30002659);
        expect(hub.tier, 'secondary');
        expect(hub.hasData, isTrue);
      });

      test('numeric fields (int daily_volume coerced to double)', () {
        expect(hub.buyPrice, 100.0);
        expect(hub.sellPrice, 110.0);
        expect(hub.spreadPercent, 10.0);
        expect(hub.netMarginPercent, 3.4);
        expect(hub.netProfitPerUnit, 4.1);
        expect(hub.dailyVolume, 125000.0);
        expect(hub.liquidityScore, 72);
      });

      test('competition object', () {
        expect(hub.competition, isNotNull);
        expect(hub.competition!.changesPerHour, 42.5);
        expect(hub.competition!.source, 'baseline');
      });
    });

    group('hub row with null / missing numeric fields', () {
      late HubRow hub;
      setUp(() => hub = HubComparisonResponse.fromJson(sample).hubs[1]);

      test('null/absent numeric fields default to 0', () {
        expect(hub.buyPrice, 0);
        expect(hub.sellPrice, 0);
        expect(hub.netMarginPercent, 0);
        expect(hub.dailyVolume, 0);
      });

      test('present int field coerced to double', () {
        expect(hub.spreadPercent, 5.0);
      });

      test('live competition source preserved (int changes_per_hour)', () {
        expect(hub.competition!.source, 'live');
        expect(hub.competition!.changesPerHour, 120.0);
      });
    });

    group('no-data hub row', () {
      late HubRow hub;
      setUp(() => hub = HubComparisonResponse.fromJson(sample).hubs[2]);

      test('has_data false and competition null', () {
        expect(hub.hasData, isFalse);
        expect(hub.competition, isNull);
      });

      test('all numeric fields default to 0', () {
        expect(hub.buyPrice, 0);
        expect(hub.sellPrice, 0);
        expect(hub.spreadPercent, 0);
        expect(hub.netMarginPercent, 0);
        expect(hub.dailyVolume, 0);
        expect(hub.liquidityScore, 0);
      });
    });

    test('hubs is empty list when key absent', () {
      final resp = HubComparisonResponse.fromJson(const {
        'type_id': 34,
        'item_name': 'X',
        'best_hub_region_id': 0,
      });
      expect(resp.hubs, isEmpty);
    });
  });

  // ── ItemSearchResponse.fromJson ────────────────────────────────────────────

  group('ItemSearchResponse.fromJson', () {
    test('maps items', () {
      final resp = ItemSearchResponse.fromJson(const {
        'items': [
          {'type_id': 34, 'name': 'Tritanium', 'group_name': 'Mineral'},
          {'type_id': 35, 'name': 'Pyerite', 'group_name': 'Mineral'},
        ],
      });
      expect(resp.items.length, 2);
      expect(resp.items[0].typeId, 34);
      expect(resp.items[0].name, 'Tritanium');
      expect(resp.items[0].groupName, 'Mineral');
    });

    test('empty when items key absent', () {
      final resp = ItemSearchResponse.fromJson(const {});
      expect(resp.items, isEmpty);
    });

    test('tolerates missing group_name', () {
      final resp = ItemSearchResponse.fromJson(const {
        'items': [
          {'type_id': 34, 'name': 'Tritanium'},
        ],
      });
      expect(resp.items.single.groupName, '');
    });
  });
}

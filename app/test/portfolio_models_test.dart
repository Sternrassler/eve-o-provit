import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/portfolio_models.dart';

void main() {
  // ── PortfolioRequest.toJson ────────────────────────────────────────────────

  group('PortfolioRequest.toJson', () {
    test('maps all fields to snake_case wire format', () {
      const req = PortfolioRequest(
        regionId: 10000002,
        shipTypeId: 649,
        capital: 500000000,
        timeBudgetMin: 120,
        liquidityCapPct: 10,
        maxItemPct: 30,
        secZones: ['high'],
      );
      expect(req.toJson(), {
        'region_id': 10000002,
        'ship_type_id': 649,
        'capital': 500000000.0,
        'time_budget_min': 120,
        'liquidity_cap_pct': 10.0,
        'max_item_pct': 30.0,
        'sec_zones': ['high'],
      });
    });
  });

  // ── PortfolioResult.fromJson ───────────────────────────────────────────────

  group('PortfolioResult.fromJson', () {
    // Realistic sample: one full item, one item with missing/null numerics —
    // exercises the null/float-robust parsing.
    const Map<String, dynamic> sample = {
      'items': [
        {
          'type_id': 34,
          'name': 'Tritanium',
          'capital_used': 1.5e8,
          'units': 120000,
          'trips_per_day': 4,
          'daily_profit': 2.5e7,
          'roi_percent': 16.7,
        },
        // Item with missing / null numeric fields.
        {
          'type_id': 35,
          'name': 'Pyerite',
          'capital_used': null,
          // units absent entirely
          'trips_per_day': 2,
          'daily_profit': null,
          // roi_percent absent
        },
      ],
      'total_capital_used': 4.8e8,
      'total_daily_profit': 6.1e7,
      'time_used_min': 118,
      'diversification_score': 72,
      'skills_applied': {
        'applied': true,
        'accounting': 5,
        'broker_relations': 5,
        'sales_tax_rate': 0.025,
        'broker_fee_rate': 0.015,
      },
    };

    test('maps top-level scalar fields (int coerced to double)', () {
      final r = PortfolioResult.fromJson(sample);
      expect(r.totalCapitalUsed, 4.8e8);
      expect(r.totalDailyProfit, 6.1e7);
      expect(r.timeUsedMin, 118.0);
      expect(r.diversificationScore, 72.0);
      expect(r.items.length, 2);
      expect(r.isEmpty, isFalse);
    });

    test('maps skills_applied object (reused SkillsApplied model)', () {
      final s = PortfolioResult.fromJson(sample).skillsApplied;
      expect(s.applied, isTrue);
      expect(s.accounting, 5);
      expect(s.brokerRelations, 5);
      expect(s.salesTaxRate, 0.025);
      expect(s.brokerFeeRate, 0.015);
    });

    test('skills_applied defaults when object absent', () {
      final noSkills = Map<String, dynamic>.from(sample)
        ..remove('skills_applied');
      final s = PortfolioResult.fromJson(noSkills).skillsApplied;
      expect(s.applied, isFalse);
      expect(s.accounting, 0);
      expect(s.salesTaxRate, 0);
    });

    group('full item', () {
      late PortfolioItem item;
      setUp(() => item = PortfolioResult.fromJson(sample).items[0]);

      test('maps all fields', () {
        expect(item.typeId, 34);
        expect(item.name, 'Tritanium');
        expect(item.capitalUsed, 1.5e8);
        expect(item.units, 120000.0);
        expect(item.tripsPerDay, 4.0);
        expect(item.dailyProfit, 2.5e7);
        expect(item.roiPercent, 16.7);
      });
    });

    group('item with null / missing numerics', () {
      late PortfolioItem item;
      setUp(() => item = PortfolioResult.fromJson(sample).items[1]);

      test('null/absent numeric fields default to 0', () {
        expect(item.capitalUsed, 0);
        expect(item.units, 0);
        expect(item.dailyProfit, 0);
        expect(item.roiPercent, 0);
      });

      test('present int field coerced to double', () {
        expect(item.tripsPerDay, 2.0);
      });
    });

    test('empty items list is a valid result (no viable portfolio)', () {
      final r = PortfolioResult.fromJson(const {
        'items': [],
        'total_capital_used': 0,
        'total_daily_profit': 0,
        'time_used_min': 0,
        'diversification_score': 0,
      });
      expect(r.items, isEmpty);
      expect(r.isEmpty, isTrue);
      expect(r.totalDailyProfit, 0);
    });

    test('items defaults to empty when key absent + scalars default to 0', () {
      final r = PortfolioResult.fromJson(const {});
      expect(r.items, isEmpty);
      expect(r.isEmpty, isTrue);
      expect(r.totalCapitalUsed, 0);
      expect(r.diversificationScore, 0);
      expect(r.skillsApplied.applied, isFalse);
    });
  });
}

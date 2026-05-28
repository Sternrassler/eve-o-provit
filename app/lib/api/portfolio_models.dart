/// Portfolio Optimizer DTOs — field names map exactly to the backend json
/// tags for POST /api/v1/trading/portfolio/optimize.
///
/// CRITICAL: parsing is null/float-robust (mirrors hub_comparison_models.dart /
/// trading_models.dart). Every numeric field tolerates absent/null values and
/// accepts any [num] (int or double). A past production bug was caused by
/// non-robust mobile DTO parsing.
library;

import 'hub_comparison_models.dart' show SkillsApplied;

export 'hub_comparison_models.dart' show SkillsApplied;

// ---------------------------------------------------------------------------
// PortfolioRequest
// ---------------------------------------------------------------------------

/// Request body for POST /api/v1/trading/portfolio/optimize.
///
/// Mirrors the backend request struct exactly (snake_case json tags). The
/// [secZones] list contains any of "high" | "low" | "null".
class PortfolioRequest {
  const PortfolioRequest({
    required this.regionId,
    required this.shipTypeId,
    required this.capital,
    required this.timeBudgetMin,
    required this.liquidityCapPct,
    required this.maxItemPct,
    required this.secZones,
  });

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `ship_type_id`
  final int shipTypeId;

  /// Backend: `capital` — total ISK available to allocate.
  final double capital;

  /// Backend: `time_budget_min` — available trading time per day, in minutes.
  final int timeBudgetMin;

  /// Backend: `liquidity_cap_pct` — max % of an item's daily volume to consume.
  final double liquidityCapPct;

  /// Backend: `max_item_pct` — max % of total capital allocated to one item.
  final double maxItemPct;

  /// Backend: `sec_zones` — security zones to include ("high"|"low"|"null").
  final List<String> secZones;

  Map<String, dynamic> toJson() => {
        'region_id': regionId,
        'ship_type_id': shipTypeId,
        'capital': capital,
        'time_budget_min': timeBudgetMin,
        'liquidity_cap_pct': liquidityCapPct,
        'max_item_pct': maxItemPct,
        'sec_zones': secZones,
      };
}

// ---------------------------------------------------------------------------
// PortfolioItem
// ---------------------------------------------------------------------------

/// A single allocated item in the suggested portfolio.
/// Backend: an element of the `items` array.
class PortfolioItem {
  const PortfolioItem({
    required this.typeId,
    required this.name,
    required this.capitalUsed,
    required this.units,
    required this.tripsPerDay,
    required this.dailyProfit,
    required this.roiPercent,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `name`
  final String name;

  /// Backend: `capital_used` — ISK allocated to this item.
  final double capitalUsed;

  /// Backend: `units` — number of units to trade.
  final double units;

  /// Backend: `trips_per_day` — round trips per day for this item.
  final double tripsPerDay;

  /// Backend: `daily_profit` — ISK profit per day from this item.
  final double dailyProfit;

  /// Backend: `roi_percent` — return on invested capital, in percent.
  final double roiPercent;

  factory PortfolioItem.fromJson(Map<String, dynamic> json) {
    return PortfolioItem(
      typeId: (json['type_id'] as num?)?.toInt() ?? 0,
      name: json['name'] as String? ?? '',
      capitalUsed: (json['capital_used'] as num?)?.toDouble() ?? 0,
      units: (json['units'] as num?)?.toDouble() ?? 0,
      tripsPerDay: (json['trips_per_day'] as num?)?.toDouble() ?? 0,
      dailyProfit: (json['daily_profit'] as num?)?.toDouble() ?? 0,
      roiPercent: (json['roi_percent'] as num?)?.toDouble() ?? 0,
    );
  }
}

// ---------------------------------------------------------------------------
// PortfolioResult
// ---------------------------------------------------------------------------

/// Response from POST /api/v1/trading/portfolio/optimize.
///
/// [items] is pre-sorted by the backend. An empty [items] list is a valid,
/// successful result meaning "no viable portfolio for this budget" — it is NOT
/// an error.
class PortfolioResult {
  const PortfolioResult({
    required this.items,
    required this.totalCapitalUsed,
    required this.totalDailyProfit,
    required this.timeUsedMin,
    required this.diversificationScore,
    required this.skillsApplied,
  });

  /// Backend: `items` — pre-sorted allocation rows.
  final List<PortfolioItem> items;

  /// Backend: `total_capital_used`
  final double totalCapitalUsed;

  /// Backend: `total_daily_profit`
  final double totalDailyProfit;

  /// Backend: `time_used_min` — minutes of the time budget actually used.
  final double timeUsedMin;

  /// Backend: `diversification_score` — 0-100.
  final double diversificationScore;

  /// Backend: `skills_applied`
  final SkillsApplied skillsApplied;

  /// True when the optimizer found no viable allocation.
  bool get isEmpty => items.isEmpty;

  factory PortfolioResult.fromJson(Map<String, dynamic> json) {
    final rawItems = json['items'] as List<dynamic>? ?? const [];
    final skillsRaw = json['skills_applied'];
    return PortfolioResult(
      items: rawItems
          .whereType<Map<String, dynamic>>()
          .map(PortfolioItem.fromJson)
          .toList(),
      totalCapitalUsed: (json['total_capital_used'] as num?)?.toDouble() ?? 0,
      totalDailyProfit: (json['total_daily_profit'] as num?)?.toDouble() ?? 0,
      timeUsedMin: (json['time_used_min'] as num?)?.toDouble() ?? 0,
      diversificationScore:
          (json['diversification_score'] as num?)?.toDouble() ?? 0,
      skillsApplied: skillsRaw is Map<String, dynamic>
          ? SkillsApplied.fromJson(skillsRaw)
          : const SkillsApplied(
              applied: false,
              accounting: 0,
              brokerRelations: 0,
              salesTaxRate: 0,
              brokerFeeRate: 0,
            ),
    );
  }
}

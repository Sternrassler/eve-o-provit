/// Multi-Hub Comparison DTOs — field names map exactly to the backend json
/// tags for POST /api/v1/trading/hubs/compare.
///
/// CRITICAL: parsing is null/float-robust (mirrors trading_models.dart). A past
/// production bug was caused by non-robust mobile DTO parsing, so every numeric
/// field tolerates absent/null values and accepts any [num] (int or double).
library;

// ---------------------------------------------------------------------------
// CompetitionInfo
// ---------------------------------------------------------------------------

/// Per-hub competition indicator.
/// Backend: `competition` object on each hub row.
class CompetitionInfo {
  const CompetitionInfo({
    required this.changesPerHour,
    required this.source,
  });

  /// Backend: `changes_per_hour` — order changes/hour (proxy for competition).
  final double changesPerHour;

  /// Backend: `source` — "live" | "baseline".
  final String source;

  factory CompetitionInfo.fromJson(Map<String, dynamic> json) {
    return CompetitionInfo(
      changesPerHour: (json['changes_per_hour'] as num?)?.toDouble() ?? 0,
      source: json['source'] as String? ?? 'baseline',
    );
  }
}

// ---------------------------------------------------------------------------
// SkillsApplied
// ---------------------------------------------------------------------------

/// Whether/how trading skills were applied to the net-margin computation.
/// Backend: `skills_applied` object.
class SkillsApplied {
  const SkillsApplied({
    required this.applied,
    required this.accounting,
    required this.brokerRelations,
    required this.salesTaxRate,
    required this.brokerFeeRate,
  });

  /// Backend: `applied`
  final bool applied;

  /// Backend: `accounting` — skill level 0-5.
  final int accounting;

  /// Backend: `broker_relations` — skill level 0-5.
  final int brokerRelations;

  /// Backend: `sales_tax_rate` — effective fractional rate (e.g. 0.025).
  final double salesTaxRate;

  /// Backend: `broker_fee_rate` — effective fractional rate (e.g. 0.015).
  final double brokerFeeRate;

  factory SkillsApplied.fromJson(Map<String, dynamic> json) {
    return SkillsApplied(
      applied: json['applied'] as bool? ?? false,
      accounting: (json['accounting'] as num?)?.toInt() ?? 0,
      brokerRelations: (json['broker_relations'] as num?)?.toInt() ?? 0,
      salesTaxRate: (json['sales_tax_rate'] as num?)?.toDouble() ?? 0,
      brokerFeeRate: (json['broker_fee_rate'] as num?)?.toDouble() ?? 0,
    );
  }
}

// ---------------------------------------------------------------------------
// HubRow
// ---------------------------------------------------------------------------

/// A single hub's station-trading metrics for the compared item.
/// Backend: an element of the `hubs` array.
///
/// When [hasData] is false the numeric fields are absent/null on the wire and
/// default to 0 here; the UI must gate display on [hasData].
class HubRow {
  const HubRow({
    required this.regionId,
    required this.regionName,
    required this.hubName,
    required this.systemId,
    required this.tier,
    required this.hasData,
    required this.buyPrice,
    required this.sellPrice,
    required this.spreadPercent,
    required this.netMarginPercent,
    required this.netProfitPerUnit,
    required this.dailyVolume,
    required this.liquidityScore,
    this.competition,
  });

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `region_name`
  final String regionName;

  /// Backend: `hub_name`
  final String hubName;

  /// Backend: `system_id`
  final int systemId;

  /// Backend: `tier` — e.g. "primary" | "secondary".
  final String tier;

  /// Backend: `has_data` — false when the hub has no market data.
  final bool hasData;

  /// Backend: `buy_price`
  final double buyPrice;

  /// Backend: `sell_price`
  final double sellPrice;

  /// Backend: `spread_percent`
  final double spreadPercent;

  /// Backend: `net_margin_percent` — skill-adjusted net margin.
  final double netMarginPercent;

  /// Backend: `net_profit_per_unit`
  final double netProfitPerUnit;

  /// Backend: `daily_volume`
  final double dailyVolume;

  /// Backend: `liquidity_score` — 0-100.
  final int liquidityScore;

  /// Backend: `competition` (object) — null when absent.
  final CompetitionInfo? competition;

  factory HubRow.fromJson(Map<String, dynamic> json) {
    final compRaw = json['competition'];
    return HubRow(
      regionId: (json['region_id'] as num?)?.toInt() ?? 0,
      regionName: json['region_name'] as String? ?? '',
      hubName: json['hub_name'] as String? ?? '',
      systemId: (json['system_id'] as num?)?.toInt() ?? 0,
      tier: json['tier'] as String? ?? '',
      hasData: json['has_data'] as bool? ?? false,
      buyPrice: (json['buy_price'] as num?)?.toDouble() ?? 0,
      sellPrice: (json['sell_price'] as num?)?.toDouble() ?? 0,
      spreadPercent: (json['spread_percent'] as num?)?.toDouble() ?? 0,
      netMarginPercent: (json['net_margin_percent'] as num?)?.toDouble() ?? 0,
      netProfitPerUnit: (json['net_profit_per_unit'] as num?)?.toDouble() ?? 0,
      dailyVolume: (json['daily_volume'] as num?)?.toDouble() ?? 0,
      liquidityScore: (json['liquidity_score'] as num?)?.toInt() ?? 0,
      competition: compRaw is Map<String, dynamic>
          ? CompetitionInfo.fromJson(compRaw)
          : null,
    );
  }
}

// ---------------------------------------------------------------------------
// HubComparisonResponse
// ---------------------------------------------------------------------------

/// Response from POST /api/v1/trading/hubs/compare.
///
/// [hubs] is pre-sorted by the backend (best net margin first, no-data rows
/// last). [bestHubRegionId] identifies the recommended hub.
class HubComparisonResponse {
  const HubComparisonResponse({
    required this.typeId,
    required this.itemName,
    required this.bestHubRegionId,
    required this.skillsApplied,
    required this.hubs,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `item_name`
  final String itemName;

  /// Backend: `best_hub_region_id` — region id of the recommended hub.
  final int bestHubRegionId;

  /// Backend: `skills_applied`
  final SkillsApplied skillsApplied;

  /// Backend: `hubs` — pre-sorted hub rows.
  final List<HubRow> hubs;

  factory HubComparisonResponse.fromJson(Map<String, dynamic> json) {
    final rawHubs = json['hubs'] as List<dynamic>? ?? const [];
    final skillsRaw = json['skills_applied'];
    return HubComparisonResponse(
      typeId: (json['type_id'] as num?)?.toInt() ?? 0,
      itemName: json['item_name'] as String? ?? '',
      bestHubRegionId: (json['best_hub_region_id'] as num?)?.toInt() ?? 0,
      skillsApplied: skillsRaw is Map<String, dynamic>
          ? SkillsApplied.fromJson(skillsRaw)
          : const SkillsApplied(
              applied: false,
              accounting: 0,
              brokerRelations: 0,
              salesTaxRate: 0,
              brokerFeeRate: 0,
            ),
      hubs: rawHubs
          .whereType<Map<String, dynamic>>()
          .map(HubRow.fromJson)
          .toList(),
    );
  }
}

// ---------------------------------------------------------------------------
// ItemSearchResult / ItemSearchResponse
// ---------------------------------------------------------------------------

/// A single item-search hit.
/// Backend: an element of the `items` array from GET /api/v1/items/search.
class ItemSearchResult {
  const ItemSearchResult({
    required this.typeId,
    required this.name,
    required this.groupName,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `name`
  final String name;

  /// Backend: `group_name`
  final String groupName;

  factory ItemSearchResult.fromJson(Map<String, dynamic> json) {
    return ItemSearchResult(
      typeId: (json['type_id'] as num?)?.toInt() ?? 0,
      name: json['name'] as String? ?? '',
      groupName: json['group_name'] as String? ?? '',
    );
  }
}

/// Response wrapper for GET /api/v1/items/search.
class ItemSearchResponse {
  const ItemSearchResponse({required this.items});

  /// Backend: `items`
  final List<ItemSearchResult> items;

  factory ItemSearchResponse.fromJson(Map<String, dynamic> json) {
    final raw = json['items'] as List<dynamic>? ?? const [];
    return ItemSearchResponse(
      items: raw
          .whereType<Map<String, dynamic>>()
          .map(ItemSearchResult.fromJson)
          .toList(),
    );
  }
}

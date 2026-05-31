/// Mining Ore-Ranking DTOs — field names map exactly to the backend
/// json tags for POST /api/v1/mining/ore-ranking.
///
/// CRITICAL: parsing is null/float-robust (mirrors hauling_models.dart /
/// hub_comparison_models.dart). Every numeric field tolerates absent/null
/// values and accepts any [num] (int or double).
library;

// ---------------------------------------------------------------------------
// OreRankingRequest
// ---------------------------------------------------------------------------

/// Request body for POST /api/v1/mining/ore-ranking.
///
/// Mirrors the backend request struct exactly (snake_case json tags).
/// [regionId] of 0 ⇒ the backend uses the character's current region.
class OreRankingRequest {
  const OreRankingRequest({
    required this.regionId,
    required this.secBand,
  });

  /// Backend: `region_id` — 0 means "use the character's current region".
  final int regionId;

  /// Backend: `sec_band` — "high" | "low" | "null".
  final String secBand;

  Map<String, dynamic> toJson() => {
        'region_id': regionId,
        'sec_band': secBand,
      };
}

// ---------------------------------------------------------------------------
// SellLocation
// ---------------------------------------------------------------------------

/// Where an item (raw ore or a refined mineral) can be sold.
/// Backend: the `raw_sell` object and each material's `sell` object.
class SellLocation {
  const SellLocation({
    required this.isStructure,
    this.stationName,
    this.systemName,
  });

  /// Backend: `is_structure` — true when the location is a player citadel
  /// (the SDE can't name it; the UI shows "Player-Struktur").
  final bool isStructure;

  /// Backend: `station_name` — nullable; absent for citadels.
  final String? stationName;

  /// Backend: `system_name` — nullable; absent for citadels.
  final String? systemName;

  factory SellLocation.fromJson(Map<String, dynamic> json) {
    return SellLocation(
      isStructure: json['is_structure'] as bool? ?? false,
      stationName: json['station_name'] as String?,
      systemName: json['system_name'] as String?,
    );
  }
}

// ---------------------------------------------------------------------------
// RefineMaterial
// ---------------------------------------------------------------------------

/// One mineral produced by reprocessing the ore, with where to sell it.
/// Backend: an element of an ore row's `materials` array.
class RefineMaterial {
  const RefineMaterial({
    required this.materialTypeId,
    required this.materialName,
    required this.effectiveQty,
    required this.buyPrice,
    required this.sell,
  });

  /// Backend: `material_type_id`
  final int materialTypeId;

  /// Backend: `material_name`
  final String materialName;

  /// Backend: `effective_qty` — units produced after the station's net yield.
  final int effectiveQty;

  /// Backend: `buy_price` — highest buy order for the mineral.
  final double buyPrice;

  /// Backend: `sell` — where the mineral fetches that best buy price.
  final SellLocation sell;

  factory RefineMaterial.fromJson(Map<String, dynamic> json) {
    return RefineMaterial(
      materialTypeId: (json['material_type_id'] as num?)?.toInt() ?? 0,
      materialName: json['material_name'] as String? ?? '',
      effectiveQty: (json['effective_qty'] as num?)?.toInt() ?? 0,
      buyPrice: (json['buy_price'] as num?)?.toDouble() ?? 0,
      sell: json['sell'] is Map<String, dynamic>
          ? SellLocation.fromJson(json['sell'] as Map<String, dynamic>)
          : const SellLocation(isStructure: false),
    );
  }
}

// ---------------------------------------------------------------------------
// OreRankRow
// ---------------------------------------------------------------------------

/// A single ore type in the ranking result.
/// Backend: an element of the `rows` array.
class OreRankRow {
  const OreRankRow({
    required this.oreTypeId,
    required this.oreName,
    required this.miningM3PerHour,
    required this.rawIskPerHour,
    required this.refineIskPerHour,
    required this.rawNetPerM3,
    required this.refineNetPerM3,
    required this.best,
    required this.deltaIskPerHour,
    required this.bestStationTax,
    this.bestStationId,
    this.bestStationName,
    this.bestStationSystem,
    this.rawSell,
    this.materials = const [],
  });

  /// Backend: `ore_type_id`
  final int oreTypeId;

  /// Backend: `ore_name`
  final String oreName;

  /// Backend: `mining_m3_per_hour`
  final double miningM3PerHour;

  /// Backend: `raw_isk_per_hour`
  final double rawIskPerHour;

  /// Backend: `refine_isk_per_hour`
  final double refineIskPerHour;

  /// Backend: `raw_net_per_m3`
  final double rawNetPerM3;

  /// Backend: `refine_net_per_m3`
  final double refineNetPerM3;

  /// Backend: `best` — "raw" | "refine"
  final String best;

  /// Backend: `delta_isk_per_hour` — difference between best and alternative.
  final double deltaIskPerHour;

  /// Backend: `best_station_id` — nullable (int?).
  final int? bestStationId;

  /// Backend: `best_station_tax`
  final double bestStationTax;

  /// Backend: `best_station_name` — nullable; the reprocess station's name.
  final String? bestStationName;

  /// Backend: `best_station_system` — nullable; the reprocess station's system.
  final String? bestStationSystem;

  /// Backend: `raw_sell` — nullable; where to sell the raw ore (best buy).
  final SellLocation? rawSell;

  /// Backend: `materials` — per-mineral breakdown for the refine path. Empty
  /// for the raw path (or when the backend omits it).
  final List<RefineMaterial> materials;

  factory OreRankRow.fromJson(Map<String, dynamic> json) {
    final rawMaterials = json['materials'] as List<dynamic>? ?? const [];
    return OreRankRow(
      oreTypeId: (json['ore_type_id'] as num?)?.toInt() ?? 0,
      oreName: json['ore_name'] as String? ?? '',
      miningM3PerHour: (json['mining_m3_per_hour'] as num?)?.toDouble() ?? 0,
      rawIskPerHour: (json['raw_isk_per_hour'] as num?)?.toDouble() ?? 0,
      refineIskPerHour:
          (json['refine_isk_per_hour'] as num?)?.toDouble() ?? 0,
      rawNetPerM3: (json['raw_net_per_m3'] as num?)?.toDouble() ?? 0,
      refineNetPerM3: (json['refine_net_per_m3'] as num?)?.toDouble() ?? 0,
      best: json['best'] as String? ?? 'raw',
      deltaIskPerHour: (json['delta_isk_per_hour'] as num?)?.toDouble() ?? 0,
      bestStationId: (json['best_station_id'] as num?)?.toInt(),
      bestStationTax: (json['best_station_tax'] as num?)?.toDouble() ?? 0,
      bestStationName: json['best_station_name'] as String?,
      bestStationSystem: json['best_station_system'] as String?,
      rawSell: json['raw_sell'] is Map<String, dynamic>
          ? SellLocation.fromJson(json['raw_sell'] as Map<String, dynamic>)
          : null,
      materials: rawMaterials
          .whereType<Map<String, dynamic>>()
          .map(RefineMaterial.fromJson)
          .toList(),
    );
  }
}

// ---------------------------------------------------------------------------
// OreRankingResponse
// ---------------------------------------------------------------------------

/// Response from POST /api/v1/mining/ore-ranking.
///
/// [rows] is pre-sorted by the backend (best_isk_per_hour desc). An empty
/// [rows] list combined with [noMiningSetup] = true indicates the character
/// has no mining lasers fitted; in that case the UI shows a visible notice.
class OreRankingResponse {
  const OreRankingResponse({
    required this.regionId,
    required this.secBand,
    required this.noMiningSetup,
    required this.rows,
  });

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `sec_band`
  final String secBand;

  /// Backend: `no_mining_setup` — true when the character has no mining lasers.
  final bool noMiningSetup;

  /// Backend: `rows` — pre-sorted by best ISK/h desc.
  final List<OreRankRow> rows;

  /// True when the ranked rows list is empty.
  bool get isEmpty => rows.isEmpty;

  factory OreRankingResponse.fromJson(Map<String, dynamic> json) {
    final rawRows = json['rows'] as List<dynamic>? ?? const [];
    return OreRankingResponse(
      regionId: (json['region_id'] as num?)?.toInt() ?? 0,
      secBand: json['sec_band'] as String? ?? 'high',
      noMiningSetup: json['no_mining_setup'] as bool? ?? false,
      rows: rawRows
          .whereType<Map<String, dynamic>>()
          .map(OreRankRow.fromJson)
          .toList(),
    );
  }
}

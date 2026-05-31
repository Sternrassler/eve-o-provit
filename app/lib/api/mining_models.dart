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

  factory OreRankRow.fromJson(Map<String, dynamic> json) {
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

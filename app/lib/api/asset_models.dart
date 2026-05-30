/// Sell-from-Assets DTOs — field names map exactly to the backend json tags
/// for GET /api/v1/trading/assets and POST /api/v1/trading/assets/sell-options.
///
/// CRITICAL: parsing is null/float-robust (mirrors hauling_models.dart /
/// hub_comparison_models.dart). Every numeric field tolerates absent/null
/// values and accepts any [num] (int or double). A past production bug was
/// caused by non-robust mobile DTO parsing.
library;

import 'hub_comparison_models.dart' show SkillsApplied;

export 'hub_comparison_models.dart' show SkillsApplied;

// ---------------------------------------------------------------------------
// AssetItem
// ---------------------------------------------------------------------------

/// A single item the character owns, from GET /api/v1/trading/assets.
/// Backend: an element of the `assets` array.
///
/// When [marketable] is false the item cannot be sold via this feature; the UI
/// must still show it but disabled in the picker.
class AssetItem {
  const AssetItem({
    required this.typeId,
    required this.name,
    required this.quantity,
    required this.locationId,
    required this.locationName,
    required this.systemId,
    required this.regionId,
    required this.marketable,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `name`
  final String name;

  /// Backend: `quantity` — units owned in this stack.
  final int quantity;

  /// Backend: `location_id` — station/structure where the stack sits.
  final int locationId;

  /// Backend: `location_name`
  final String locationName;

  /// Backend: `system_id` — solar system of the stack's location.
  final int systemId;

  /// Backend: `region_id` — region of the stack's location.
  final int regionId;

  /// Backend: `marketable` — false ⇒ cannot be sold (disabled in picker).
  final bool marketable;

  factory AssetItem.fromJson(Map<String, dynamic> json) {
    return AssetItem(
      typeId: (json['type_id'] as num?)?.toInt() ?? 0,
      name: json['name'] as String? ?? '',
      quantity: (json['quantity'] as num?)?.toInt() ?? 0,
      locationId: (json['location_id'] as num?)?.toInt() ?? 0,
      locationName: json['location_name'] as String? ?? '',
      systemId: (json['system_id'] as num?)?.toInt() ?? 0,
      regionId: (json['region_id'] as num?)?.toInt() ?? 0,
      marketable: json['marketable'] as bool? ?? false,
    );
  }
}

// ---------------------------------------------------------------------------
// AssetsResponse
// ---------------------------------------------------------------------------

/// Response from GET /api/v1/trading/assets.
class AssetsResponse {
  const AssetsResponse({
    required this.assets,
    required this.count,
    this.cacheExpiresAt,
  });

  /// Backend: `assets`
  final List<AssetItem> assets;

  /// Backend: `count`
  final int count;

  /// Backend: `cache_expires_at` — ESI serves the same snapshot until this
  /// point; a refresh before then returns identical data. Null when ESI didn't
  /// return a parseable `Expires` header.
  final DateTime? cacheExpiresAt;

  /// True when the character owns no (marketable or otherwise) assets.
  bool get isEmpty => assets.isEmpty;

  factory AssetsResponse.fromJson(Map<String, dynamic> json) {
    final rawAssets = json['assets'] as List<dynamic>? ?? const [];
    return AssetsResponse(
      assets: rawAssets
          .whereType<Map<String, dynamic>>()
          .map(AssetItem.fromJson)
          .toList(),
      count: (json['count'] as num?)?.toInt() ?? 0,
      cacheExpiresAt: DateTime.tryParse(
        json['cache_expires_at'] as String? ?? '',
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// SellOptionsRequest
// ---------------------------------------------------------------------------

/// Request body for POST /api/v1/trading/assets/sell-options.
/// Mirrors the backend request struct exactly (snake_case json tags).
class SellOptionsRequest {
  const SellOptionsRequest({
    required this.typeId,
    required this.locationId,
    required this.quantity,
    required this.avoidLowSec,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `location_id` — the asset stack's current location (origin).
  final int locationId;

  /// Backend: `quantity` — units to sell.
  final int quantity;

  /// Backend: `avoid_low_sec` — when true, low-sec hops are excluded.
  final bool avoidLowSec;

  Map<String, dynamic> toJson() => {
        'type_id': typeId,
        'location_id': locationId,
        'quantity': quantity,
        'avoid_low_sec': avoidLowSec,
      };
}

// ---------------------------------------------------------------------------
// SellOption
// ---------------------------------------------------------------------------

/// A single ranked sell location for the chosen asset.
/// Backend: an element of the `options` array (pre-sorted by total_net desc).
///
/// When [hasData] is false there is no buy order / route ("kein Marktpreis");
/// the numeric fields default to 0 and the UI must render the option muted.
class SellOption {
  const SellOption({
    required this.scope,
    required this.regionId,
    required this.regionName,
    required this.stationId,
    required this.stationName,
    required this.systemName,
    required this.buyPrice,
    required this.unitNet,
    required this.totalNet,
    required this.iskPerHour,
    required this.jumps,
    required this.travelTimeMin,
    required this.securityRisk,
    required this.hasData,
  });

  /// Backend: `scope` — "hub" | "current_region".
  final String scope;

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `region_name`
  final String regionName;

  /// Backend: `station_id` — used as the autopilot waypoint destination.
  final int stationId;

  /// Backend: `station_name`
  final String stationName;

  /// Backend: `system_name`
  final String systemName;

  /// Backend: `buy_price` — best buy-order price at this station.
  final double buyPrice;

  /// Backend: `unit_net` — per-unit net ISK after sales tax.
  final double unitNet;

  /// Backend: `total_net` — total net ISK for the full quantity (headline).
  final double totalNet;

  /// Backend: `isk_per_hour` — net ISK per hour of travel; 0 for local sales.
  /// Drives the option ranking.
  final double iskPerHour;

  /// Backend: `jumps` — number of jumps from the asset's origin system.
  final int jumps;

  /// Backend: `travel_time_min` — estimated travel time in minutes.
  final double travelTimeMin;

  /// Backend: `security_risk` — "safe" | "caution" | "danger".
  final String securityRisk;

  /// Backend: `has_data` — false ⇒ no market price / route (render muted).
  final bool hasData;

  /// True when this option represents one of the major trade hubs.
  bool get isHub => scope == 'hub';

  factory SellOption.fromJson(Map<String, dynamic> json) {
    return SellOption(
      scope: json['scope'] as String? ?? 'current_region',
      regionId: (json['region_id'] as num?)?.toInt() ?? 0,
      regionName: json['region_name'] as String? ?? '',
      stationId: (json['station_id'] as num?)?.toInt() ?? 0,
      stationName: json['station_name'] as String? ?? '',
      systemName: json['system_name'] as String? ?? '',
      buyPrice: (json['buy_price'] as num?)?.toDouble() ?? 0,
      unitNet: (json['unit_net'] as num?)?.toDouble() ?? 0,
      totalNet: (json['total_net'] as num?)?.toDouble() ?? 0,
      iskPerHour: (json['isk_per_hour'] as num?)?.toDouble() ?? 0,
      jumps: (json['jumps'] as num?)?.toInt() ?? 0,
      travelTimeMin: (json['travel_time_min'] as num?)?.toDouble() ?? 0,
      securityRisk: json['security_risk'] as String? ?? 'safe',
      hasData: json['has_data'] as bool? ?? false,
    );
  }
}

// ---------------------------------------------------------------------------
// SellOptionsResponse
// ---------------------------------------------------------------------------

/// Response from POST /api/v1/trading/assets/sell-options.
///
/// [options] is pre-sorted by the backend (total_net desc). [best] is the top
/// actionable option, or null when there is none. An empty [options] list is a
/// valid, successful "no sell locations found" result — it is NOT an error.
class SellOptionsResponse {
  const SellOptionsResponse({
    required this.typeId,
    required this.name,
    required this.quantity,
    required this.originSystemId,
    required this.best,
    required this.options,
    required this.skillsApplied,
    this.notRoutableReason,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `name`
  final String name;

  /// Backend: `quantity`
  final int quantity;

  /// Backend: `origin_system_id`
  final int originSystemId;

  /// Backend: `best` — top actionable option, or null.
  final SellOption? best;

  /// Backend: `options` — pre-sorted by total_net desc.
  final List<SellOption> options;

  /// Backend: `skills_applied`
  final SkillsApplied skillsApplied;

  /// Backend: `not_routable_reason` — set when the empty result has a known
  /// cause (currently only `"origin_in_player_structure"`). UI shows an
  /// actionable hint instead of the generic "no buyers" message.
  final String? notRoutableReason;

  /// True when no sell locations were returned.
  bool get isEmpty => options.isEmpty;

  factory SellOptionsResponse.fromJson(Map<String, dynamic> json) {
    final rawOptions = json['options'] as List<dynamic>? ?? const [];
    final bestRaw = json['best'];
    final skillsRaw = json['skills_applied'];
    return SellOptionsResponse(
      typeId: (json['type_id'] as num?)?.toInt() ?? 0,
      name: json['name'] as String? ?? '',
      quantity: (json['quantity'] as num?)?.toInt() ?? 0,
      originSystemId: (json['origin_system_id'] as num?)?.toInt() ?? 0,
      best: bestRaw is Map<String, dynamic>
          ? SellOption.fromJson(bestRaw)
          : null,
      options: rawOptions
          .whereType<Map<String, dynamic>>()
          .map(SellOption.fromJson)
          .toList(),
      skillsApplied: skillsRaw is Map<String, dynamic>
          ? SkillsApplied.fromJson(skillsRaw)
          : const SkillsApplied(
              applied: false,
              accounting: 0,
              brokerRelations: 0,
              salesTaxRate: 0,
              brokerFeeRate: 0,
            ),
      notRoutableReason: json['not_routable_reason'] as String?,
    );
  }
}

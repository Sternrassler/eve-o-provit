/// Neighborhood Hauling Routes DTOs — field names map exactly to the backend
/// json tags for POST /api/v1/trading/hauling/routes.
///
/// CRITICAL: parsing is null/float-robust (mirrors hub_comparison_models.dart /
/// portfolio_models.dart). Every numeric field tolerates absent/null values and
/// accepts any [num] (int or double). A past production bug was caused by
/// non-robust mobile DTO parsing.
library;

import 'hub_comparison_models.dart' show SkillsApplied;

export 'hub_comparison_models.dart' show SkillsApplied;

// ---------------------------------------------------------------------------
// HaulingRequest
// ---------------------------------------------------------------------------

/// Request body for POST /api/v1/trading/hauling/routes.
///
/// Mirrors the backend request struct exactly (snake_case json tags).
/// [originRegionId] of 0 ⇒ the backend uses the character's current region.
class HaulingRequest {
  const HaulingRequest({
    required this.originRegionId,
    required this.shipTypeId,
    required this.capital,
    required this.avoidLowSec,
    required this.maxRoutes,
  });

  /// Backend: `origin_region_id` — 0 means "use the character's current region".
  final int originRegionId;

  /// Backend: `ship_type_id`
  final int shipTypeId;

  /// Backend: `capital` — total ISK available for hauling.
  final double capital;

  /// Backend: `avoid_low_sec` — when true, low-sec hops are excluded.
  final bool avoidLowSec;

  /// Backend: `max_routes` — cap on the number of routes returned.
  final int maxRoutes;

  Map<String, dynamic> toJson() => {
        'origin_region_id': originRegionId,
        'ship_type_id': shipTypeId,
        'capital': capital,
        'avoid_low_sec': avoidLowSec,
        'max_routes': maxRoutes,
      };
}

// ---------------------------------------------------------------------------
// HaulingItem
// ---------------------------------------------------------------------------

/// A single item in a hauling route's optimal cargo load.
/// Backend: an element of a route's `items` array.
class HaulingItem {
  const HaulingItem({
    required this.typeId,
    required this.name,
    required this.quantity,
    required this.buyPrice,
    required this.sellPrice,
    required this.unitVolume,
    required this.profit,
    required this.profitPerM3,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `name`
  final String name;

  /// Backend: `quantity` — units to haul.
  final int quantity;

  /// Backend: `buy_price` — per-unit buy price at the origin station.
  final double buyPrice;

  /// Backend: `sell_price` — per-unit sell price at the destination station.
  final double sellPrice;

  /// Backend: `unit_volume` — m³ per unit.
  final double unitVolume;

  /// Backend: `profit` — total net profit for this item across [quantity].
  final double profit;

  /// Backend: `profit_per_m3` — net profit per m³ of cargo.
  final double profitPerM3;

  factory HaulingItem.fromJson(Map<String, dynamic> json) {
    return HaulingItem(
      typeId: (json['type_id'] as num?)?.toInt() ?? 0,
      name: json['name'] as String? ?? '',
      quantity: (json['quantity'] as num?)?.toInt() ?? 0,
      buyPrice: (json['buy_price'] as num?)?.toDouble() ?? 0,
      sellPrice: (json['sell_price'] as num?)?.toDouble() ?? 0,
      unitVolume: (json['unit_volume'] as num?)?.toDouble() ?? 0,
      profit: (json['profit'] as num?)?.toDouble() ?? 0,
      profitPerM3: (json['profit_per_m3'] as num?)?.toDouble() ?? 0,
    );
  }
}

// ---------------------------------------------------------------------------
// HaulingRoute
// ---------------------------------------------------------------------------

/// A single profitable station→station hauling route with an optimal load.
/// Backend: an element of the `routes` array.
class HaulingRoute {
  const HaulingRoute({
    required this.buySystemName,
    required this.buyStationName,
    required this.buyStationId,
    required this.sellSystemName,
    required this.sellStationName,
    required this.sellStationId,
    required this.jumps,
    required this.travelTimeMin,
    required this.securityRisk,
    required this.totalProfit,
    required this.totalCapital,
    required this.totalVolume,
    required this.iskPerHour,
    required this.items,
  });

  /// Backend: `buy_system_name`
  final String buySystemName;

  /// Backend: `buy_station_name`
  final String buyStationName;

  /// Backend: `buy_station_id` — station/structure id used as buy waypoint.
  final int buyStationId;

  /// Backend: `sell_system_name`
  final String sellSystemName;

  /// Backend: `sell_station_name`
  final String sellStationName;

  /// Backend: `sell_station_id` — station/structure id used as sell waypoint.
  final int sellStationId;

  /// Backend: `jumps` — number of jumps from buy to sell station.
  final int jumps;

  /// Backend: `travel_time_min` — estimated travel time in minutes.
  final double travelTimeMin;

  /// Backend: `security_risk` — "safe" | "caution" | "danger".
  final String securityRisk;

  /// Backend: `total_profit` — net ISK profit for the full optimal load.
  final double totalProfit;

  /// Backend: `total_capital` — ISK required to buy the optimal load.
  final double totalCapital;

  /// Backend: `total_volume` — m³ of the optimal load.
  final double totalVolume;

  /// Backend: `isk_per_hour` — the route's headline efficiency metric.
  final double iskPerHour;

  /// Backend: `items` — the optimal cargo load.
  final List<HaulingItem> items;

  factory HaulingRoute.fromJson(Map<String, dynamic> json) {
    final rawItems = json['items'] as List<dynamic>? ?? const [];
    return HaulingRoute(
      buySystemName: json['buy_system_name'] as String? ?? '',
      buyStationName: json['buy_station_name'] as String? ?? '',
      buyStationId: (json['buy_station_id'] as num?)?.toInt() ?? 0,
      sellSystemName: json['sell_system_name'] as String? ?? '',
      sellStationName: json['sell_station_name'] as String? ?? '',
      sellStationId: (json['sell_station_id'] as num?)?.toInt() ?? 0,
      jumps: (json['jumps'] as num?)?.toInt() ?? 0,
      travelTimeMin: (json['travel_time_min'] as num?)?.toDouble() ?? 0,
      securityRisk: json['security_risk'] as String? ?? 'safe',
      totalProfit: (json['total_profit'] as num?)?.toDouble() ?? 0,
      totalCapital: (json['total_capital'] as num?)?.toDouble() ?? 0,
      totalVolume: (json['total_volume'] as num?)?.toDouble() ?? 0,
      iskPerHour: (json['isk_per_hour'] as num?)?.toDouble() ?? 0,
      items: rawItems
          .whereType<Map<String, dynamic>>()
          .map(HaulingItem.fromJson)
          .toList(),
    );
  }
}

// ---------------------------------------------------------------------------
// HaulingResponse
// ---------------------------------------------------------------------------

/// Response from POST /api/v1/trading/hauling/routes.
///
/// [routes] is pre-sorted by the backend (isk_per_hour desc). An empty
/// [routes] list is a valid, successful result meaning "no profitable routes
/// in the neighborhood" — it is NOT an error.
class HaulingResponse {
  const HaulingResponse({
    required this.originRegionId,
    required this.regionsScanned,
    required this.routes,
    required this.skillsApplied,
  });

  /// Backend: `origin_region_id` — the region actually used as origin.
  final int originRegionId;

  /// Backend: `regions_scanned` — region ids included in the scan.
  final List<int> regionsScanned;

  /// Backend: `routes` — pre-sorted by isk_per_hour desc.
  final List<HaulingRoute> routes;

  /// Backend: `skills_applied`
  final SkillsApplied skillsApplied;

  /// True when no profitable routes were found.
  bool get isEmpty => routes.isEmpty;

  factory HaulingResponse.fromJson(Map<String, dynamic> json) {
    final rawRoutes = json['routes'] as List<dynamic>? ?? const [];
    final rawRegions = json['regions_scanned'] as List<dynamic>? ?? const [];
    final skillsRaw = json['skills_applied'];
    return HaulingResponse(
      originRegionId: (json['origin_region_id'] as num?)?.toInt() ?? 0,
      regionsScanned: rawRegions
          .whereType<num>()
          .map((n) => n.toInt())
          .toList(),
      routes: rawRoutes
          .whereType<Map<String, dynamic>>()
          .map(HaulingRoute.fromJson)
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
    );
  }
}

/// Trading DTOs — field names map exactly to the backend json tags in
/// backend/internal/models/trading.go and models/api_responses.go.
library;

// ---------------------------------------------------------------------------
// VolumeMetrics
// ---------------------------------------------------------------------------

/// Market volume and liquidity metrics for an item.
/// Backend: models.VolumeMetrics (trading.go).
class VolumeMetrics {
  const VolumeMetrics({
    required this.typeId,
    required this.regionId,
    required this.dailyVolumeAvg,
    required this.dailyIskTurnover,
    required this.liquidityScore,
    required this.dataDays,
  });

  /// Backend: `type_id`
  final int typeId;

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `daily_volume_avg` — 30-day average daily volume
  final double dailyVolumeAvg;

  /// Backend: `daily_isk_turnover` — average daily ISK traded
  final double dailyIskTurnover;

  /// Backend: `liquidity_score` — 0-100
  final int liquidityScore;

  /// Backend: `data_days` — days of historical data available
  final int dataDays;

  factory VolumeMetrics.fromJson(Map<String, dynamic> json) {
    return VolumeMetrics(
      typeId: json['type_id'] as int,
      regionId: json['region_id'] as int,
      dailyVolumeAvg: (json['daily_volume_avg'] as num).toDouble(),
      dailyIskTurnover: (json['daily_isk_turnover'] as num).toDouble(),
      liquidityScore: json['liquidity_score'] as int,
      dataDays: json['data_days'] as int,
    );
  }
}

// ---------------------------------------------------------------------------
// TradingRoute
// ---------------------------------------------------------------------------

/// A profitable trading route returned by the backend.
/// Backend: models.TradingRoute (trading.go).
class TradingRoute {
  const TradingRoute({
    required this.itemTypeId,
    required this.itemName,
    required this.buySystemId,
    required this.buySystemName,
    required this.buyStationId,
    required this.buyStationName,
    required this.buyPrice,
    required this.sellSystemId,
    required this.sellSystemName,
    required this.sellStationId,
    required this.sellStationName,
    required this.sellPrice,
    required this.buySecurityStatus,
    required this.sellSecurityStatus,
    required this.minRouteSecurityStatus,
    required this.quantity,
    required this.profitPerUnit,
    required this.totalProfit,
    required this.spreadPercent,
    required this.travelTimeSeconds,
    required this.roundTripSeconds,
    required this.iskPerHour,
    required this.jumps,
    required this.itemVolume,
    required this.numberOfTours,
    required this.profitPerTour,
    required this.totalTimeMinutes,
    required this.baseTravelTimeSeconds,
    required this.skilledTravelTimeSeconds,
    required this.baseIskPerHour,
    required this.timeImprovementPercent,
    required this.buyBrokerFee,
    required this.sellBrokerFee,
    required this.brokerFees,
    required this.salesTax,
    required this.estimatedRelistFee,
    required this.totalFees,
    required this.grossProfit,
    required this.grossMarginPercent,
    required this.netProfit,
    required this.netProfitPercent,
    required this.cargoUsed,
    required this.cargoCapacity,
    required this.cargoUtilization,
    required this.baseCargoCapacity,
    required this.skillBonusPercent,
    required this.fittingBonusM3,
    required this.totalInvestment,
    this.volumeMetrics,
    this.liquidationDays,
    this.dailyProfit,
  });

  /// Backend: `item_type_id`
  final int itemTypeId;

  /// Backend: `item_name`
  final String itemName;

  /// Backend: `buy_system_id`
  final int buySystemId;

  /// Backend: `buy_system_name`
  final String buySystemName;

  /// Backend: `buy_station_id`
  final int buyStationId;

  /// Backend: `buy_station_name`
  final String buyStationName;

  /// Backend: `buy_price`
  final double buyPrice;

  /// Backend: `sell_system_id`
  final int sellSystemId;

  /// Backend: `sell_system_name`
  final String sellSystemName;

  /// Backend: `sell_station_id`
  final int sellStationId;

  /// Backend: `sell_station_name`
  final String sellStationName;

  /// Backend: `sell_price`
  final double sellPrice;

  /// Backend: `buy_security_status`
  final double buySecurityStatus;

  /// Backend: `sell_security_status`
  final double sellSecurityStatus;

  /// Backend: `min_route_security_status`
  final double minRouteSecurityStatus;

  /// Backend: `quantity`
  final int quantity;

  /// Backend: `profit_per_unit`
  final double profitPerUnit;

  /// Backend: `total_profit`
  final double totalProfit;

  /// Backend: `spread_percent`
  final double spreadPercent;

  /// Backend: `travel_time_seconds`
  final double travelTimeSeconds;

  /// Backend: `round_trip_seconds`
  final double roundTripSeconds;

  /// Backend: `isk_per_hour`
  final double iskPerHour;

  /// Backend: `jumps`
  final int jumps;

  /// Backend: `item_volume`
  final double itemVolume;

  /// Backend: `number_of_tours`
  final int numberOfTours;

  /// Backend: `profit_per_tour`
  final double profitPerTour;

  /// Backend: `total_time_minutes`
  final double totalTimeMinutes;

  /// Backend: `base_travel_time_seconds`
  final double baseTravelTimeSeconds;

  /// Backend: `skilled_travel_time_seconds`
  final double skilledTravelTimeSeconds;

  /// Backend: `base_isk_per_hour`
  final double baseIskPerHour;

  /// Backend: `time_improvement_percent`
  final double timeImprovementPercent;

  /// Backend: `buy_broker_fee`
  final double buyBrokerFee;

  /// Backend: `sell_broker_fee`
  final double sellBrokerFee;

  /// Backend: `broker_fees`
  final double brokerFees;

  /// Backend: `sales_tax`
  final double salesTax;

  /// Backend: `estimated_relist_fee`
  final double estimatedRelistFee;

  /// Backend: `total_fees`
  final double totalFees;

  /// Backend: `gross_profit`
  final double grossProfit;

  /// Backend: `gross_margin_percent`
  final double grossMarginPercent;

  /// Backend: `net_profit`
  final double netProfit;

  /// Backend: `net_profit_percent`
  final double netProfitPercent;

  /// Backend: `cargo_used`
  final double cargoUsed;

  /// Backend: `cargo_capacity`
  final double cargoCapacity;

  /// Backend: `cargo_utilization`
  final double cargoUtilization;

  /// Backend: `base_cargo_capacity`
  final double baseCargoCapacity;

  /// Backend: `skill_bonus_percent`
  final double skillBonusPercent;

  /// Backend: `fitting_bonus_m3`
  final double fittingBonusM3;

  /// Backend: `total_investment`
  final double totalInvestment;

  /// Backend: `volume_metrics` (omitempty — nullable)
  final VolumeMetrics? volumeMetrics;

  /// Backend: `liquidation_days` (omitempty — nullable)
  final double? liquidationDays;

  /// Backend: `daily_profit` (omitempty — nullable)
  final double? dailyProfit;

  factory TradingRoute.fromJson(Map<String, dynamic> json) {
    final vmRaw = json['volume_metrics'];
    return TradingRoute(
      itemTypeId: json['item_type_id'] as int,
      itemName: json['item_name'] as String,
      buySystemId: (json['buy_system_id'] as num).toInt(),
      buySystemName: json['buy_system_name'] as String,
      buyStationId: (json['buy_station_id'] as num).toInt(),
      buyStationName: json['buy_station_name'] as String,
      buyPrice: (json['buy_price'] as num).toDouble(),
      sellSystemId: (json['sell_system_id'] as num).toInt(),
      sellSystemName: json['sell_system_name'] as String,
      sellStationId: (json['sell_station_id'] as num).toInt(),
      sellStationName: json['sell_station_name'] as String,
      sellPrice: (json['sell_price'] as num).toDouble(),
      buySecurityStatus: (json['buy_security_status'] as num).toDouble(),
      sellSecurityStatus: (json['sell_security_status'] as num).toDouble(),
      minRouteSecurityStatus:
          (json['min_route_security_status'] as num).toDouble(),
      quantity: json['quantity'] as int,
      profitPerUnit: (json['profit_per_unit'] as num).toDouble(),
      totalProfit: (json['total_profit'] as num).toDouble(),
      spreadPercent: (json['spread_percent'] as num).toDouble(),
      travelTimeSeconds: (json['travel_time_seconds'] as num).toDouble(),
      roundTripSeconds: (json['round_trip_seconds'] as num).toDouble(),
      iskPerHour: (json['isk_per_hour'] as num).toDouble(),
      jumps: json['jumps'] as int,
      itemVolume: (json['item_volume'] as num).toDouble(),
      numberOfTours: json['number_of_tours'] as int,
      profitPerTour: (json['profit_per_tour'] as num).toDouble(),
      totalTimeMinutes: (json['total_time_minutes'] as num).toDouble(),
      baseTravelTimeSeconds:
          (json['base_travel_time_seconds'] as num).toDouble(),
      skilledTravelTimeSeconds:
          (json['skilled_travel_time_seconds'] as num).toDouble(),
      baseIskPerHour: (json['base_isk_per_hour'] as num).toDouble(),
      timeImprovementPercent:
          (json['time_improvement_percent'] as num).toDouble(),
      buyBrokerFee: (json['buy_broker_fee'] as num).toDouble(),
      sellBrokerFee: (json['sell_broker_fee'] as num).toDouble(),
      brokerFees: (json['broker_fees'] as num).toDouble(),
      salesTax: (json['sales_tax'] as num).toDouble(),
      estimatedRelistFee: (json['estimated_relist_fee'] as num).toDouble(),
      totalFees: (json['total_fees'] as num).toDouble(),
      grossProfit: (json['gross_profit'] as num).toDouble(),
      grossMarginPercent: (json['gross_margin_percent'] as num).toDouble(),
      netProfit: (json['net_profit'] as num).toDouble(),
      netProfitPercent: (json['net_profit_percent'] as num).toDouble(),
      cargoUsed: (json['cargo_used'] as num).toDouble(),
      cargoCapacity: (json['cargo_capacity'] as num).toDouble(),
      cargoUtilization: (json['cargo_utilization'] as num).toDouble(),
      baseCargoCapacity: (json['base_cargo_capacity'] as num).toDouble(),
      skillBonusPercent: (json['skill_bonus_percent'] as num).toDouble(),
      fittingBonusM3: (json['fitting_bonus_m3'] as num).toDouble(),
      totalInvestment: (json['total_investment'] as num).toDouble(),
      volumeMetrics: vmRaw != null
          ? VolumeMetrics.fromJson(vmRaw as Map<String, dynamic>)
          : null,
      liquidationDays: json['liquidation_days'] != null
          ? (json['liquidation_days'] as num).toDouble()
          : null,
      dailyProfit: json['daily_profit'] != null
          ? (json['daily_profit'] as num).toDouble()
          : null,
    );
  }
}

// ---------------------------------------------------------------------------
// RouteCalculationRequest
// ---------------------------------------------------------------------------

/// Request body for POST /api/v1/trading/routes/calculate.
/// Backend: models.RouteCalculationRequest (trading.go).
class RouteCalculationRequest {
  const RouteCalculationRequest({
    required this.regionId,
    required this.shipTypeId,
    this.cargoCapacity,
    this.warpSpeed,
    this.alignTime,
    this.minDailyVolume,
    this.maxLiquidationDays,
    this.includeVolumeMetrics,
  });

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `ship_type_id`
  final int shipTypeId;

  /// Backend: `cargo_capacity` (omitempty — nullable)
  final double? cargoCapacity;

  /// Backend: `warp_speed` (omitempty — nullable)
  final double? warpSpeed;

  /// Backend: `align_time` (omitempty — nullable)
  final double? alignTime;

  /// Backend: `min_daily_volume` (omitempty — nullable)
  final double? minDailyVolume;

  /// Backend: `max_liquidation_days` (omitempty — nullable)
  final double? maxLiquidationDays;

  /// Backend: `include_volume_metrics` (omitempty — nullable)
  final bool? includeVolumeMetrics;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{
      'region_id': regionId,
      'ship_type_id': shipTypeId,
    };
    if (cargoCapacity != null) m['cargo_capacity'] = cargoCapacity;
    if (warpSpeed != null) m['warp_speed'] = warpSpeed;
    if (alignTime != null) m['align_time'] = alignTime;
    if (minDailyVolume != null) m['min_daily_volume'] = minDailyVolume;
    if (maxLiquidationDays != null) {
      m['max_liquidation_days'] = maxLiquidationDays;
    }
    if (includeVolumeMetrics != null) {
      m['include_volume_metrics'] = includeVolumeMetrics;
    }
    return m;
  }
}

// ---------------------------------------------------------------------------
// RouteCalculationResponse
// ---------------------------------------------------------------------------

/// Response from POST /api/v1/trading/routes/calculate.
/// Backend: models.RouteCalculationResponse (trading.go).
class RouteCalculationResponse {
  const RouteCalculationResponse({
    required this.regionId,
    required this.regionName,
    required this.shipTypeId,
    required this.shipName,
    required this.cargoCapacity,
    required this.calculationTimeMs,
    required this.routes,
    this.warning,
  });

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `region_name`
  final String regionName;

  /// Backend: `ship_type_id`
  final int shipTypeId;

  /// Backend: `ship_name`
  final String shipName;

  /// Backend: `cargo_capacity`
  final double cargoCapacity;

  /// Backend: `calculation_time_ms`
  final int calculationTimeMs;

  /// Backend: `routes`
  final List<TradingRoute> routes;

  /// Backend: `warning` (omitempty — nullable)
  final String? warning;

  factory RouteCalculationResponse.fromJson(Map<String, dynamic> json) {
    final rawRoutes = json['routes'] as List<dynamic>? ?? [];
    return RouteCalculationResponse(
      regionId: json['region_id'] as int,
      regionName: json['region_name'] as String,
      shipTypeId: json['ship_type_id'] as int,
      shipName: json['ship_name'] as String,
      cargoCapacity: (json['cargo_capacity'] as num).toDouble(),
      calculationTimeMs: (json['calculation_time_ms'] as num).toInt(),
      routes: rawRoutes
          .map((e) => TradingRoute.fromJson(e as Map<String, dynamic>))
          .toList(),
      // Backend uses omitempty; absent key → null
      warning: json['warning'] as String?,
    );
  }
}

// ---------------------------------------------------------------------------
// Region
// ---------------------------------------------------------------------------

/// An EVE Online region.
/// Backend: models.Region (trading.go); GET /api/v1/sde/regions returns
/// models.RegionsResponse { regions: []Region, count: int }.
class Region {
  const Region({required this.id, required this.name});

  /// Backend: `id`
  final int id;

  /// Backend: `name`
  final String name;

  factory Region.fromJson(Map<String, dynamic> json) {
    return Region(
      id: (json['id'] as num).toInt(),
      name: json['name'] as String,
    );
  }
}

// ---------------------------------------------------------------------------
// RegionsResponse
// ---------------------------------------------------------------------------

/// Response wrapper for GET /api/v1/sde/regions.
/// Backend: models.RegionsResponse (trading.go).
class RegionsResponse {
  const RegionsResponse({required this.regions, required this.count});

  /// Backend: `regions`
  final List<Region> regions;

  /// Backend: `count`
  final int count;

  factory RegionsResponse.fromJson(Map<String, dynamic> json) {
    final rawList = json['regions'] as List<dynamic>? ?? [];
    return RegionsResponse(
      regions: rawList
          .map((e) => Region.fromJson(e as Map<String, dynamic>))
          .toList(),
      count: json['count'] as int,
    );
  }
}

// ---------------------------------------------------------------------------
// MarketDataStalenessResponse
// ---------------------------------------------------------------------------

/// Response from GET /api/v1/market/staleness/:region.
/// Backend: models.MarketDataStalenessResponse (api_responses.go).
class MarketDataStalenessResponse {
  const MarketDataStalenessResponse({
    required this.regionId,
    required this.regionName,
    required this.lastUpdate,
    required this.ageMinutes,
    required this.status,
    required this.refreshAllowed,
  });

  /// Backend: `region_id`
  final int regionId;

  /// Backend: `region_name`
  final String regionName;

  /// Backend: `last_update`
  final DateTime lastUpdate;

  /// Backend: `age_minutes`
  final int ageMinutes;

  /// Backend: `status` — "fresh" | "stale" | "very_stale"
  final String status;

  /// Backend: `refresh_allowed`
  final bool refreshAllowed;

  factory MarketDataStalenessResponse.fromJson(Map<String, dynamic> json) {
    return MarketDataStalenessResponse(
      regionId: (json['region_id'] as num).toInt(),
      regionName: json['region_name'] as String,
      lastUpdate: DateTime.parse(json['last_update'] as String),
      ageMinutes: json['age_minutes'] as int,
      status: json['status'] as String,
      refreshAllowed: json['refresh_allowed'] as bool,
    );
  }
}

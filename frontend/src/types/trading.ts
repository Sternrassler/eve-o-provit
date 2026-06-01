// types/trading.ts

export type LoadingState = "idle" | "loading" | "success" | "error";

export interface TradingRoute {
  rank?: number; // Optional for API responses
  item_type_id: number;
  item_name: string;
  // API fields
  buy_system_id?: number;
  buy_system_name?: string;
  buy_station_id?: number;
  buy_station_name?: string;
  sell_system_id?: number;
  sell_system_name?: string;
  sell_station_id?: number;
  sell_station_name?: string;
  buy_security_status?: number; // Security status of buy system (0.0 - 1.0)
  sell_security_status?: number; // Security status of sell system (0.0 - 1.0)
  min_route_security_status?: number; // Minimum security of all systems on route
  // Mock data fields (for backward compatibility)
  origin_system_id?: number;
  origin_system_name?: string;
  destination_system_id?: number;
  destination_system_name?: string;
  quantity: number;
  buy_price: number;
  sell_price: number;
  total_investment?: number;
  total_revenue?: number;
  profit?: number; // Mock data
  total_profit?: number; // API field
  spread_percent: number;
  travel_time_seconds: number;
  round_trip_seconds: number;
  isk_per_hour: number;
  jumps?: number; // API field
  item_volume?: number; // API field
  profit_per_unit?: number; // API field
  // Multi-tour fields
  number_of_tours?: number;
  profit_per_tour?: number;
  total_time_minutes?: number;
  // Navigation Skills fields
  base_travel_time_seconds?: number; // Travel time without navigation skills
  skilled_travel_time_seconds?: number; // Travel time with navigation skills applied
  base_isk_per_hour?: number; // ISK/h without navigation skills
  time_improvement_percent?: number; // Percentage improvement from skills
  // Cargo fields
  cargo_used?: number; // m³ actually used
  cargo_capacity?: number; // Total effective capacity (with skills + fitting)
  cargo_utilization?: number; // Percentage 0-100
  base_cargo_capacity?: number; // Base capacity without skills
  skill_bonus_percent?: number; // Total skill bonus %
  fitting_bonus_m3?: number; // Fitting modules bonus (absolute m³)
  // Fee fields
  gross_profit?: number; // Revenue - Investment (before fees)
  gross_margin_percent?: number; // (gross_profit / total_investment) * 100
  sales_tax?: number; // Sales tax fee
  broker_fees?: number; // Broker fees
  estimated_relist_fee?: number; // Estimated relist fee
  total_fees?: number; // Sum of all fees
  net_profit?: number; // Gross profit - total fees
  net_profit_percent?: number; // Net margin percentage
  // Volume & Liquidity fields (Issue #53)
  volume_metrics?: VolumeMetrics; // Market volume and liquidity data
  liquidation_days?: number; // Estimated days to sell inventory
  daily_profit?: number; // Profit per day (net_profit / liquidation_days)
}

export interface VolumeMetrics {
  type_id: number;
  region_id: number;
  daily_volume_avg: number; // 30-day average daily volume
  daily_isk_turnover: number; // Average daily ISK traded (volume × avg_price)
  liquidity_score: number; // 0-100 score based on volume stability
  data_days: number; // Number of days of historical data available
}

export interface Region {
  id: number;
  name: string;
}

export interface Ship {
  item_id: number;
  type_id: number;
  name: string;
  /** Base hull cargo (without skills/modules). */
  cargo_capacity: number;
  /** Effective cargo (hull + skills + fitted modules); same value the optimizer
   *  uses. Absent when the ship has no cargo-expander fitting — then the base
   *  `cargo_capacity` ("Basis") is the correct value. */
  effective_cargo_capacity?: number;
  /** True when the backend's fitting enrichment ERRORED for this ship (ESI/
   *  network), so the effective volume is genuinely unknown. The UI shows a
   *  visible "unknown" marker rather than passing off the base hull as fitted. */
  effective_cargo_unavailable?: boolean;
}

export interface TradingFilters {
  minSpread: number;
  minProfit: number;
  maxTravelTime: number;
  allowHighSec: boolean;
  allowLowSec: boolean;
  allowNullSec: boolean;
  // Volume filters (Issue #53)
  minDailyVolume?: number; // Minimum daily volume (items/day)
  maxLiquidationDays?: number; // Maximum liquidation time (days)
  includeVolumeMetrics?: boolean; // Whether to include volume metrics
}

export interface ItemSearchResult {
  type_id: number;
  name: string;
  group_name: string;
}

// Multi-Hub Comparison (Issue #43)

export interface CompetitionInfo {
  changes_per_hour: number;
  source: "live" | "baseline" | string;
}

export interface SkillsApplied {
  applied: boolean;
  accounting: number;
  broker_relations: number;
  sales_tax_rate: number;
  broker_fee_rate: number;
}

export interface HubRow {
  region_id: number;
  region_name: string;
  hub_name: string;
  system_id: number;
  tier: string;
  has_data: boolean;
  buy_price: number;
  sell_price: number;
  spread_percent: number;
  net_margin_percent: number;
  net_profit_per_unit: number;
  daily_volume: number;
  liquidity_score: number;
  competition: CompetitionInfo;
}

export interface HubComparisonResult {
  type_id: number;
  item_name: string;
  best_hub_region_id: number;
  skills_applied: SkillsApplied;
  hubs: HubRow[];
}

// ROI Calculator & Capital Allocation Optimizer (Issue #44)

export interface PortfolioRequest {
  region_id: number;
  ship_type_id: number;
  capital: number;
  time_budget_min: number;
  liquidity_cap_pct: number;
  max_item_pct: number;
  sec_zones: string[]; // e.g. ["high", "low", "null"]
  /** Selected ship instance's effective cargo (m³). When set, the optimizer
   *  uses this exact value instead of recomputing per ship type. Omitted when
   *  the instance's effective cargo is unavailable/unknown (backend falls back
   *  to the per-type recompute). */
  cargo_capacity?: number;
}

export interface PortfolioItem {
  type_id: number;
  name: string;
  capital_used: number;
  units: number;
  trips_per_day: number;
  daily_profit: number;
  roi_percent: number;
  /** Daily profit per hour of haul time. Items are pre-sorted by this metric. */
  isk_per_hour: number;
  // Buy/sell location (station → station haul)
  buy_system_name: string;
  buy_station_name: string;
  buy_station_id: number;
  sell_system_name: string;
  sell_station_name: string;
  sell_station_id: number;
}

export interface PortfolioResult {
  items: PortfolioItem[];
  total_capital_used: number;
  total_daily_profit: number;
  time_used_min: number;
  diversification_score: number;
  skills_applied: SkillsApplied;
}

// Neighborhood Hauling Routes (Issue #45)

export type SecurityRisk = "safe" | "caution" | "danger";

export interface HaulingRequest {
  origin_region_id: number; // 0 ⇒ backend uses character's current region
  ship_type_id: number;
  capital: number;
  avoid_low_sec: boolean;
  max_routes: number;
  /** Selected ship instance's effective cargo (m³). When set, the backend uses
   *  this exact value instead of recomputing per ship type. Omitted when the
   *  instance's effective cargo is unavailable/unknown (backend falls back to
   *  the per-type recompute). */
  cargo_capacity?: number;
}

export interface HaulingItem {
  type_id: number;
  name: string;
  quantity: number;
  buy_price: number;
  sell_price: number;
  unit_volume: number;
  profit: number;
  profit_per_m3: number;
}

export interface HaulingRoute {
  buy_system_name: string;
  buy_station_name: string;
  buy_station_id: number;
  sell_system_name: string;
  sell_station_name: string;
  sell_station_id: number;
  jumps: number;
  travel_time_min: number;
  security_risk: SecurityRisk;
  total_profit: number;
  total_capital: number;
  total_volume: number;
  isk_per_hour: number;
  items: HaulingItem[];
}

export interface HaulingResponse {
  origin_region_id: number;
  regions_scanned: number[];
  routes: HaulingRoute[]; // pre-sorted by isk_per_hour desc
  skills_applied: SkillsApplied;
}

export interface InventorySellRequest {
  type_id: number;
  quantity: number;
  buy_price_per_unit: number;
  region_id: number;
  min_profit_per_unit: number;
  security_filter: string; // "all", "highsec", "highlow"
}

export interface InventorySellRoute {
  sell_station_id: number;
  sell_station_name: string;
  sell_system_id: number;
  sell_system_name: string;
  sell_security_status: number;
  buy_order_price: number;
  tax_rate: number;
  net_price_per_unit: number;
  profit_per_unit: number;
  available_quantity: number;
  total_profit: number;
  route_jumps: number;
  route_system_ids: number[];
  min_route_security_status: number;
}

// Sell-from-Assets (Issue #107)

export interface AssetItem {
  type_id: number;
  name: string;
  quantity: number;
  location_id: number;
  location_name: string;
  system_id: number;
  region_id: number;
  marketable: boolean;
}

export interface AssetsResponse {
  assets: AssetItem[];
  count: number;
  /** ESI cache-expiry (RFC 3339). Until this point ESI serves the same
   *  snapshot, so a refresh before then returns identical data. Absent when
   *  ESI didn't return a parseable Expires header. */
  cache_expires_at?: string;
}

export interface SellOptionsRequest {
  type_id: number;
  location_id: number;
  quantity: number;
  avoid_low_sec: boolean;
}

export type SellOptionScope = "hub" | "current_region";

export interface SellOption {
  scope: SellOptionScope;
  region_id: number;
  region_name: string;
  station_id: number;
  station_name: string;
  system_name: string;
  buy_price: number;
  unit_net: number;
  total_net: number;
  /** Net ISK per hour of travel. 0 for local sales (no travel) — they rank
   *  ahead of any remote sale by `total_net`. */
  isk_per_hour: number;
  jumps: number;
  travel_time_min: number;
  security_risk: SecurityRisk;
  has_data: boolean;
}

export interface SellOptionsResponse {
  type_id: number;
  name: string;
  quantity: number;
  origin_system_id: number;
  best: SellOption | null;
  options: SellOption[]; // pre-sorted by total_net desc
  skills_applied: SkillsApplied;
  /** Set when the empty options list deserves an explanation. Current values:
   *  "origin_in_player_structure" — the item sits in a citadel; SDE can't
   *  resolve its system, so no route is possible. */
  not_routable_reason?: string;
}

// Mining Ore Ranking (ore ISK/hour calculator)

export interface OreRankingRequest {
  region_id: number;
  sec_band: string; // "high" | "low" | "null"
}

export interface SellLocation {
  station_name?: string;
  system_name?: string;
  is_structure: boolean;
  location_id?: number;
}

export interface RefineMaterial {
  material_type_id: number;
  material_name: string;
  effective_qty: number;
  buy_price: number;
  sell: SellLocation;
}

export interface OreRankRow {
  ore_type_id: number;
  ore_name: string;
  mining_m3_per_hour: number;
  raw_isk_per_hour: number;
  refine_isk_per_hour: number;
  raw_net_per_m3: number;
  refine_net_per_m3: number;
  best: string; // "raw" | "refine"
  delta_isk_per_hour: number;
  best_station_id?: number;
  best_station_tax: number;
  best_station_name?: string;
  best_station_system?: string;
  raw_sell?: SellLocation;
  materials?: RefineMaterial[];
  hull_yield_multiplier?: number;
  crystal_multiplier?: number;
  is_estimate?: boolean;
  estimate_reason?: string;
  effective_isk_per_hour?: number;
  load_volume_m3?: number;
  fill_minutes?: number;
  cycle_minutes?: number;
  route_jumps?: number;
  sell_system_name?: string;
}

export interface OreRankingResponse {
  region_id: number;
  sec_band: string;
  no_mining_setup: boolean;
  rows: OreRankRow[];
}

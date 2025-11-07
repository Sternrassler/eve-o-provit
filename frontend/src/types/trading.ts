// types/trading.ts
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
}

export interface Region {
  id: number;
  name: string;
}

export interface Ship {
  type_id: number;
  name: string;
  cargo_capacity: number;
}

export interface TradingFilters {
  minSpread: number;
  minProfit: number;
  maxTravelTime: number;
  allowHighSec: boolean;
  allowLowSec: boolean;
  allowNullSec: boolean;
}

export interface ItemSearchResult {
  type_id: number;
  name: string;
  group_name: string;
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

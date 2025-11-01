// types/trading.ts
export interface TradingRoute {
  rank: number;
  item_type_id: number;
  item_name: string;
  origin_system_id: number;
  origin_system_name: string;
  destination_system_id: number;
  destination_system_name: string;
  quantity: number;
  buy_price: number;
  sell_price: number;
  total_investment: number;
  total_revenue: number;
  profit: number;
  spread_percent: number;
  travel_time_seconds: number;
  round_trip_seconds: number;
  isk_per_hour: number;
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
  minVolumeFill: number;
}

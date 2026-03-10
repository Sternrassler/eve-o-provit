"use client";

import { TradingRoute } from "@/types/trading";
import { cn, formatISK } from "@/lib/utils";

interface RouteCargoSectionProps {
  route: TradingRoute;
}

export function RouteCargoSection({ route }: RouteCargoSectionProps) {
  if (route.cargo_capacity === undefined || route.cargo_used === undefined) {
    return null;
  }

  return (
    <div className="border-t pt-3">
      <div className="flex items-center justify-between text-sm mb-1.5">
        <span className="text-muted-foreground">Cargo</span>
        <span className="font-medium">{route.cargo_capacity.toFixed(0)} m³</span>
      </div>

      <div className="h-2 bg-gray-700 dark:bg-gray-800 rounded-full overflow-hidden">
        <div
          className={cn(
            "h-full transition-all",
            route.cargo_utilization !== undefined && route.cargo_utilization >= 95
              ? "bg-green-500"
              : route.cargo_utilization !== undefined && route.cargo_utilization >= 70
              ? "bg-yellow-500"
              : "bg-red-500"
          )}
          style={{ width: `${Math.min(route.cargo_utilization || 0, 100)}%` }}
        />
      </div>

      {route.cargo_utilization !== undefined && route.cargo_utilization < 70 && (
        <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-1">
          ⚠️ Niedrige Cargo-Auslastung ({route.cargo_utilization.toFixed(0)}%)
        </p>
      )}

      {route.total_investment !== undefined && (
        <div className="flex items-center justify-between text-xs text-muted-foreground mt-2 pt-2 border-t border-gray-700/50">
          <span>Einkaufspreis je Ladung:</span>
          <span className="font-medium">{formatISK(route.total_investment)}</span>
        </div>
      )}
    </div>
  );
}

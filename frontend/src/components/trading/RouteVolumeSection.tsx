"use client";

import { TradingRoute } from "@/types/trading";
import { cn, formatISK } from "@/lib/utils";

const LIQUIDATION_WARNING_DAYS = 7;
const LIQUIDATION_MEDIUM_DAYS = 14;

interface RouteVolumeSectionProps {
  route: TradingRoute;
}

export function RouteVolumeSection({ route }: RouteVolumeSectionProps) {
  if (!route.volume_metrics) return null;

  const { volume_metrics } = route;

  return (
    <div className="border-t pt-3">
      <div className="text-sm font-medium text-muted-foreground mb-2">
        Volume & Liquidität
      </div>

      <div className="grid grid-cols-2 gap-4 mb-2 text-sm">
        <div>
          <div className="text-muted-foreground">Tägliches Volume</div>
          <div className="font-medium">
            {volume_metrics.daily_volume_avg.toFixed(0)} Items
          </div>
        </div>
        <div>
          <div className="text-muted-foreground">ISK Umsatz/Tag</div>
          <div className="font-medium">
            {formatISK(volume_metrics.daily_isk_turnover)}
          </div>
        </div>
      </div>

      {route.liquidation_days !== undefined && (
        <div className="mb-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Liquidationszeit</span>
            <span
              className={cn(
                "font-medium",
                route.liquidation_days <= LIQUIDATION_WARNING_DAYS
                  ? "text-green-600 dark:text-green-400"
                  : route.liquidation_days <= LIQUIDATION_MEDIUM_DAYS
                  ? "text-yellow-600 dark:text-yellow-400"
                  : "text-red-600 dark:text-red-400"
              )}
            >
              {route.liquidation_days.toFixed(1)} Tage
              {route.liquidation_days > LIQUIDATION_WARNING_DAYS && " ⚠️"}
            </span>
          </div>
          {route.liquidation_days > LIQUIDATION_WARNING_DAYS && (
            <p className="text-xs text-muted-foreground mt-1">
              ⚠️ Niedrige Liquidität: Verkauf dauert{" "}
              {route.liquidation_days > 30
                ? "über einen Monat"
                : `${Math.round(route.liquidation_days)} Tage`}
            </p>
          )}
        </div>
      )}

      {route.daily_profit !== undefined && route.daily_profit > 0 && (
        <div className="mb-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Profit/Tag</span>
            <span className="font-bold text-primary">
              {formatISK(route.daily_profit)}
            </span>
          </div>
        </div>
      )}

      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs">
          <span className="text-muted-foreground">Liquiditätsscore</span>
          <span className="font-medium">{volume_metrics.liquidity_score}/100</span>
        </div>
        <div className="h-2 bg-gray-700 dark:bg-gray-800 rounded-full overflow-hidden">
          <div
            className={cn(
              "h-full transition-all",
              volume_metrics.liquidity_score >= 70
                ? "bg-green-500"
                : volume_metrics.liquidity_score >= 40
                ? "bg-yellow-500"
                : "bg-red-500"
            )}
            style={{ width: `${volume_metrics.liquidity_score}%` }}
          />
        </div>
      </div>

      {volume_metrics.data_days < 30 && (
        <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-2">
          ℹ️ Nur {volume_metrics.data_days} Tage Daten verfügbar
        </p>
      )}
    </div>
  );
}

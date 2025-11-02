"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TradingRoute } from "@/types/trading";
import { ArrowRight, TrendingUp } from "lucide-react";
import { cn } from "@/lib/utils";

interface TradingRouteCardProps {
  route: TradingRoute;
}

export function TradingRouteCard({ route }: TradingRouteCardProps) {
  const formatISK = (value: number) => {
    if (value >= 1000000) {
      return `${(value / 1000000).toFixed(2)}M ISK`;
    }
    if (value >= 1000) {
      return `${(value / 1000).toFixed(0)}k ISK`;
    }
    return `${value.toFixed(0)} ISK`;
  };

  const formatTime = (seconds: number) => {
    if (seconds === 0) return "Station Trading";
    const minutes = Math.round(seconds / 60);
    return `${minutes}min`;
  };

  const getSpreadColor = (spread: number) => {
    if (spread >= 10) return "text-green-600 dark:text-green-400";
    if (spread >= 5) return "text-yellow-600 dark:text-yellow-400";
    return "text-red-600 dark:text-red-400";
  };

  // Determine card background based on security status
  const getSecurityBackground = () => {
    const buySecStatus = route.buy_security_status ?? 1.0;
    const sellSecStatus = route.sell_security_status ?? 1.0;
    const minSecStatus = Math.min(buySecStatus, sellSecStatus);

    if (minSecStatus >= 0.5) {
      // High sec: light green
      return "bg-green-100/60 dark:bg-green-900/30";
    } else if (minSecStatus > 0.0) {
      // Low sec: light yellow
      return "bg-yellow-100/60 dark:bg-yellow-900/30";
    } else {
      // Null sec: light red
      return "bg-red-100/60 dark:bg-red-900/30";
    }
  };

  return (
    <Card className={cn("transition-all hover:shadow-lg hover:border-primary/50", getSecurityBackground())}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-base">
            <span className="text-muted-foreground">#{route.rank}</span>
            <span>{route.item_name}</span>
          </div>
          <div className="flex items-center gap-1 text-lg font-bold text-primary">
            <TrendingUp className="size-5" />
            {formatISK(route.isk_per_hour)}/h
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {/* Route Path */}
        <div className="space-y-1.5">
          <div className="flex items-center gap-2 text-sm">
            <span className="font-medium">{route.buy_system_name || route.origin_system_name}</span>
            <ArrowRight className="size-4 text-muted-foreground" />
            <span className="font-medium">{route.sell_system_name || route.destination_system_name}</span>
          </div>
          {(route.buy_station_name || route.sell_station_name) && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground/80">
              <span className="truncate">{route.buy_station_name}</span>
              <ArrowRight className="size-3.5" />
              <span className="truncate">{route.sell_station_name}</span>
            </div>
          )}
        </div>

        {/* Quantity */}
        <div className="text-sm text-muted-foreground">
          Menge: {route.quantity.toLocaleString("de-DE")}
        </div>

        {/* Prices */}
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <div className="text-muted-foreground">Kaufpreis</div>
            <div className="font-medium">{formatISK(route.buy_price)}</div>
          </div>
          <div>
            <div className="text-muted-foreground">Verkaufspreis</div>
            <div className="font-medium">{formatISK(route.sell_price)}</div>
          </div>
        </div>

        {/* Profit and Spread */}
        <div className="grid grid-cols-2 gap-4 border-t pt-3">
          <div>
            <div className="text-sm text-muted-foreground">Gewinn</div>
            <div className="text-lg font-bold">{formatISK(route.total_profit || route.profit || 0)}</div>
          </div>
          <div>
            <div className="text-sm text-muted-foreground">Spread</div>
            <div className={cn("text-lg font-bold", getSpreadColor(route.spread_percent))}>
              {route.spread_percent.toFixed(1)}%
            </div>
          </div>
        </div>

        {/* Travel Time */}
        <div className="border-t pt-3 text-sm">
          <div className="text-muted-foreground">Reisezeit</div>
          <div className="font-medium">{formatTime(route.travel_time_seconds)}</div>
        </div>
      </CardContent>
    </Card>
  );
}

"use client";

import { TradingRoute } from "@/types/trading";
import { TradingRouteCard } from "./TradingRouteCard";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState, useMemo } from "react";

interface TradingRouteListProps {
  routes: TradingRoute[];
  loading: boolean;
  error?: string;
  onRetry?: () => void;
}

export function TradingRouteList({
  routes,
  loading,
  error,
  onRetry,
}: TradingRouteListProps) {
  type SortOption = "isk_per_hour" | "total_profit" | "daily_profit" | "liquidation";
  const [sortBy, setSortBy] = useState<SortOption>("isk_per_hour");

  // Check if routes have volume metrics
  const hasVolumeMetrics = routes.some(r => r.volume_metrics !== undefined);

  // Sort routes based on selected criteria
  const sortedRoutes = useMemo(() => {
    if (routes.length === 0) return routes;

    return [...routes].sort((a, b) => {
      switch (sortBy) {
        case "isk_per_hour":
          return (b.isk_per_hour || 0) - (a.isk_per_hour || 0);
        case "total_profit": {
          const profitA = a.net_profit || a.total_profit || 0;
          const profitB = b.net_profit || b.total_profit || 0;
          return profitB - profitA;
        }
        case "daily_profit":
          return (b.daily_profit || 0) - (a.daily_profit || 0);
        case "liquidation":
          // Sort by shortest liquidation time (ascending)
          return (a.liquidation_days || 999) - (b.liquidation_days || 999);
        default:
          return 0;
      }
    });
  }, [routes, sortBy]);

  if (loading) {
    return (
      <div className="grid gap-6 md:grid-cols-2">
        {Array.from({ length: 10 }).map((_, i) => (
          <div key={i} className="space-y-3">
            <Skeleton className="h-[300px] w-full rounded-xl" />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="size-4" />
        <AlertTitle>Fehler</AlertTitle>
        <AlertDescription className="flex items-center gap-2">
          <span>{error}</span>
          {onRetry && (
            <Button variant="outline" size="sm" onClick={onRetry}>
              Erneut versuchen
            </Button>
          )}
        </AlertDescription>
      </Alert>
    );
  }

  if (routes.length === 0) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <div className="text-center">
          <p className="text-lg font-medium">Keine Routen gefunden</p>
          <p className="text-sm text-muted-foreground">
            Passe die Filter an oder wähle eine andere Region
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Sort Controls */}
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">Sortieren nach:</span>
        <Select value={sortBy} onValueChange={(value) => setSortBy(value as SortOption)}>
          <SelectTrigger className="w-[250px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="isk_per_hour">ISK/Stunde</SelectItem>
            <SelectItem value="total_profit">Gesamt-Profit</SelectItem>
            {hasVolumeMetrics && (
              <>
                <SelectItem value="daily_profit">Täglicher Profit ✨</SelectItem>
                <SelectItem value="liquidation">Schnellster Verkauf</SelectItem>
              </>
            )}
          </SelectContent>
        </Select>
        <span className="text-sm text-muted-foreground">
          {sortedRoutes.length} {sortedRoutes.length === 1 ? "Route" : "Routen"}
        </span>
      </div>

      {/* Route Cards */}
      <div className="grid gap-6 md:grid-cols-2">
        {sortedRoutes.map((route, index) => (
          <TradingRouteCard key={route.item_type_id || route.rank || index} route={route} />
        ))}
      </div>
    </div>
  );
}

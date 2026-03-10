"use client";

import { TradingRoute } from "@/types/trading";
import { TrendingUp, Repeat } from "lucide-react";
import { cn, formatISK } from "@/lib/utils";
import { CardTitle } from "@/components/ui/card";

interface RouteHeaderProps {
  route: TradingRoute;
}

export function RouteHeader({ route }: RouteHeaderProps) {
  const isMultiTour = route.number_of_tours && route.number_of_tours > 1;

  return (
    <CardTitle className="flex items-center justify-between">
      <div className="flex items-center gap-2 text-base">
        <span className="text-muted-foreground">#{route.rank}</span>
        <span>{route.item_name}</span>
        {isMultiTour && (
          <span className="inline-flex items-center gap-1 rounded-md bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            <Repeat className="size-3" />
            {route.number_of_tours}x
          </span>
        )}
      </div>
      <div
        className={cn(
          "flex items-center gap-1 text-lg font-bold",
          (route.net_profit ?? route.total_profit ?? 0) < 0
            ? "text-red-600 dark:text-red-400"
            : "text-primary"
        )}
      >
        <TrendingUp className="size-5" />
        {formatISK(route.isk_per_hour)}/h
      </div>
    </CardTitle>
  );
}

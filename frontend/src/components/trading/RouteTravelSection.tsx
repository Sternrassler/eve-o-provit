"use client";

import { TradingRoute } from "@/types/trading";
import { TrendingUp } from "lucide-react";

function formatTime(seconds: number): string {
  if (seconds === 0) return "Station Trading";
  return `${Math.round(seconds / 60)}min`;
}

interface RouteTravelSectionProps {
  route: TradingRoute;
}

export function RouteTravelSection({ route }: RouteTravelSectionProps) {
  const isMultiTour = route.number_of_tours && route.number_of_tours > 1;

  return (
    <>
      <div className="border-t pt-3 text-sm space-y-1">
        <div className="text-muted-foreground">Reisezeit</div>
        <div className="font-medium">
          {isMultiTour && route.total_time_minutes
            ? `${Math.round(route.total_time_minutes)} min (${route.number_of_tours} Touren)`
            : formatTime(route.round_trip_seconds)}
          {route.jumps !== undefined && (
            <span className="text-xs text-muted-foreground ml-1">
              ({route.jumps} {route.jumps === 1 ? "Jump" : "Jumps"})
            </span>
          )}
        </div>

        {route.time_improvement_percent != null &&
          route.time_improvement_percent > 0 && (
            <div className="flex items-center gap-2 text-xs">
              {route.base_travel_time_seconds &&
                route.base_travel_time_seconds > 0 && (
                  <span className="text-muted-foreground line-through">
                    {formatTime(route.base_travel_time_seconds)}
                  </span>
                )}
              <span className="text-green-600 dark:text-green-400 font-medium">
                -{route.time_improvement_percent.toFixed(1)}% mit Skills
              </span>
            </div>
          )}
      </div>

      {route.time_improvement_percent != null &&
        route.time_improvement_percent > 5 && (
          <div className="border-t pt-3 flex items-center justify-between text-sm">
            <span className="text-muted-foreground">ISK/h Boost</span>
            <span className="inline-flex items-center gap-1 rounded-md bg-green-100 dark:bg-green-900/30 px-2 py-0.5 text-xs font-medium text-green-600 dark:text-green-400">
              <TrendingUp className="size-3" />
              +{route.time_improvement_percent.toFixed(0)}% durch Nav Skills
            </span>
          </div>
        )}
    </>
  );
}

"use client";

import { TradingRoute } from "@/types/trading";
import { ArrowRight, Copy, Navigation } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface RoutePathSectionProps {
  route: TradingRoute;
  isAuthenticated: boolean;
  isSettingRoute: boolean;
  onCopyLink: () => void;
  onSetRoute: () => void;
}

export function RoutePathSection({
  route,
  isAuthenticated,
  isSettingRoute,
  onCopyLink,
  onSetRoute,
}: RoutePathSectionProps) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2 text-sm">
        <span className="font-medium">
          {route.buy_system_name || route.origin_system_name}
        </span>
        <ArrowRight className="size-4 text-muted-foreground" />
        <span className="font-medium">
          {route.sell_system_name || route.destination_system_name}
        </span>
      </div>

      {(route.buy_station_name || route.sell_station_name) && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground/80">
          <span className="truncate">{route.buy_station_name}</span>
          <ArrowRight className="size-3.5" />
          <span className="truncate">{route.sell_station_name}</span>

          {route.buy_station_id && route.sell_station_id && (
            <TooltipProvider>
              <div className="flex items-center gap-1 ml-auto">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      onClick={onCopyLink}
                      className="p-1 rounded hover:bg-muted transition-colors"
                      aria-label="EVE Chat Link kopieren"
                    >
                      <Copy className="size-3.5" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>EVE Chat Link kopieren</p>
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      onClick={onSetRoute}
                      disabled={!isAuthenticated || isSettingRoute}
                      className="p-1 rounded hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      aria-label="Route in EVE setzen"
                    >
                      <Navigation
                        className={cn(
                          "size-3.5",
                          isSettingRoute && "animate-pulse"
                        )}
                      />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>
                      {isAuthenticated
                        ? "Route in EVE setzen"
                        : "EVE Login erforderlich"}
                    </p>
                  </TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          )}
        </div>
      )}
    </div>
  );
}

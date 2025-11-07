"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TradingRoute } from "@/types/trading";
import { ArrowRight, TrendingUp, Repeat, Navigation, Copy } from "lucide-react";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/lib/auth-context";
import { useState } from "react";
import { useToast } from "@/hooks/use-toast";

interface TradingRouteCardProps {
  route: TradingRoute;
}

export function TradingRouteCard({ route }: TradingRouteCardProps) {
  const { getAuthHeader, isAuthenticated } = useAuth();
  const { toast } = useToast();
  const [isSettingRoute, setIsSettingRoute] = useState(false);

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
    // Use minimum route security if available (considers all systems on route)
    // Otherwise fall back to min of start/destination (backward compatibility)
    const minSecStatus = route.min_route_security_status ?? 
                         Math.min(route.buy_security_status ?? 1.0, route.sell_security_status ?? 1.0);

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

  const isMultiTour = route.number_of_tours && route.number_of_tours > 1;

  // Copy EVE Chat Link to clipboard
  const handleCopyLink = async () => {
    if (!route.buy_station_id || !route.sell_station_id) {
      toast({
        title: "Keine Stationen",
        description: "Route hat keine Station-IDs",
        variant: "destructive",
      });
      return;
    }

    const buyLink = `<url=showinfo:${route.buy_station_id}>${route.buy_station_name || route.buy_station_id}</url>`;
    const sellLink = `<url=showinfo:${route.sell_station_id}>${route.sell_station_name || route.sell_station_id}</url>`;
    const routeLink = `${buyLink} → ${sellLink}`;

    try {
      await navigator.clipboard.writeText(routeLink);
      toast({
        title: "Link kopiert",
        description: "EVE Chat Link in Zwischenablage kopiert",
      });
    } catch (err) {
      console.error("Failed to copy:", err);
      toast({
        title: "Fehler",
        description: "Kopieren fehlgeschlagen",
        variant: "destructive",
      });
    }
  };

  // Set route waypoints in EVE client via ESI
  const handleSetRoute = async () => {
    if (!isAuthenticated) {
      toast({
        title: "Nicht eingeloggt",
        description: "EVE SSO Login erforderlich",
        variant: "destructive",
      });
      return;
    }

    if (!route.buy_station_id || !route.sell_station_id) {
      toast({
        title: "Keine Stationen",
        description: "Route hat keine Station-IDs",
        variant: "destructive",
      });
      return;
    }

    setIsSettingRoute(true);

    try {
      const authHeader = getAuthHeader();
      if (!authHeader) {
        throw new Error("No auth token");
      }

      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

      // Set buy station waypoint (clear existing, add to beginning)
      const buyResponse = await fetch(`${apiUrl}/api/v1/esi/ui/autopilot/waypoint`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: authHeader,
        },
        body: JSON.stringify({
          destination_id: route.buy_station_id,
          clear_other_waypoints: true,
          add_to_beginning: false,
        }),
      });

      if (!buyResponse.ok) {
        if (buyResponse.status === 401 || buyResponse.status === 403) {
          throw new Error("Missing scope esi-ui.write_waypoint.v1");
        }
        if (buyResponse.status === 404) {
          throw new Error("EVE client not running");
        }
        throw new Error(`Failed to set buy waypoint: ${buyResponse.statusText}`);
      }

      // Set sell station waypoint (append)
      const sellResponse = await fetch(`${apiUrl}/api/v1/esi/ui/autopilot/waypoint`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: authHeader,
        },
        body: JSON.stringify({
          destination_id: route.sell_station_id,
          clear_other_waypoints: false,
          add_to_beginning: false,
        }),
      });

      if (!sellResponse.ok) {
        if (sellResponse.status === 404) {
          throw new Error("EVE client not running");
        }
        throw new Error(`Failed to set sell waypoint: ${sellResponse.statusText}`);
      }

      toast({
        title: "Route gesetzt",
        description: `Waypoints in EVE gesetzt: ${route.buy_station_name} → ${route.sell_station_name}`,
      });
    } catch (err) {
      console.error("Failed to set route:", err);
      const message = err instanceof Error ? err.message : "Unbekannter Fehler";
      toast({
        title: "Fehler",
        description: message,
        variant: "destructive",
      });
    } finally {
      setIsSettingRoute(false);
    }
  };

  return (
    <Card className={cn("transition-all hover:shadow-lg hover:border-primary/50", getSecurityBackground())}>
      <CardHeader>
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
              
              {/* Route Action Buttons */}
              {route.buy_station_id && route.sell_station_id && (
                <TooltipProvider>
                  <div className="flex items-center gap-1 ml-auto">
                    {/* Copy Link Button */}
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          onClick={handleCopyLink}
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

                    {/* Set Route Button */}
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          onClick={handleSetRoute}
                          disabled={!isAuthenticated || isSettingRoute}
                          className="p-1 rounded hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          aria-label="Route in EVE setzen"
                        >
                          <Navigation className={cn("size-3.5", isSettingRoute && "animate-pulse")} />
                        </button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>{isAuthenticated ? "Route in EVE setzen" : "EVE Login erforderlich"}</p>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                </TooltipProvider>
              )}
            </div>
          )}
        </div>

        {/* Quantity */}
        <div className="text-sm text-muted-foreground">
          Menge: {route.quantity.toLocaleString("de-DE")}
          {isMultiTour && route.number_of_tours && (
            <span className="block sm:inline">
              {" "}(≈{Math.ceil(route.quantity / route.number_of_tours).toLocaleString("de-DE")} pro Tour)
            </span>
          )}
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

        {/* Multi-tour Profit Display */}
        {isMultiTour ? (
          <div className="space-y-2 border-t pt-3">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-sm text-muted-foreground">Pro Tour</div>
                <div className="font-bold">{formatISK(route.profit_per_tour || 0)}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Gesamt</div>
                <div className="text-lg font-bold">{formatISK(route.total_profit || route.profit || 0)}</div>
              </div>
            </div>
          </div>
        ) : (
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
        )}

        {/* Spread for multi-tour (moved below profit) */}
        {isMultiTour && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Spread</span>
            <span className={cn("font-bold", getSpreadColor(route.spread_percent))}>
              {route.spread_percent.toFixed(1)}%
            </span>
          </div>
        )}

        {/* Travel Time with Skills Improvement */}
        <div className="border-t pt-3 text-sm space-y-1">
          <div className="text-muted-foreground">Reisezeit</div>
          <div className="font-medium">
            {isMultiTour && route.total_time_minutes
              ? `${Math.round(route.total_time_minutes)} min (${route.number_of_tours} Touren)`
              : formatTime(route.travel_time_seconds)}
          </div>
          
          {/* Show skills improvement if available */}
          {route.time_improvement_percent && route.time_improvement_percent > 0 && (
            <div className="flex items-center gap-2 text-xs">
              {route.base_travel_time_seconds && (
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

        {/* ISK/h Improvement Badge */}
        {route.time_improvement_percent && route.time_improvement_percent > 5 && (
          <div className="border-t pt-3 flex items-center justify-between text-sm">
            <span className="text-muted-foreground">ISK/h Boost</span>
            <span className="inline-flex items-center gap-1 rounded-md bg-green-100 dark:bg-green-900/30 px-2 py-0.5 text-xs font-medium text-green-600 dark:text-green-400">
              <TrendingUp className="size-3" />
              +{route.time_improvement_percent.toFixed(0)}% durch Nav Skills
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

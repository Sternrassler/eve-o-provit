"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TradingRoute } from "@/types/trading";
import { ArrowRight, TrendingUp, Repeat, Navigation, Copy, Info } from "lucide-react";
import { cn, formatISKWithSeparators } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/lib/auth-context";
import { useState } from "react";
import { useToast } from "@/hooks/use-toast";
import { FeeBreakdown } from "./FeeBreakdown";

// Liquidation thresholds for color coding and warnings
const LIQUIDATION_WARNING_DAYS = 7;
const LIQUIDATION_MEDIUM_DAYS = 14;

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

  const getProfitColor = (netMarginPercent: number) => {
    if (netMarginPercent >= 10) return "text-green-600 dark:text-green-400";
    if (netMarginPercent >= 5) return "text-yellow-600 dark:text-yellow-400";
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
          {route.item_volume && (
            <span className="text-xs ml-1">
              ({route.item_volume} m³/Einheit, {(route.quantity * route.item_volume).toFixed(0)} m³ gesamt)
            </span>
          )}
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

        {/* Fee Breakdown and Profit Display */}
        {route.gross_profit !== undefined && route.net_profit !== undefined ? (
          <div className="space-y-3 border-t pt-3">
            {/* Gross Profit Section */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-sm text-muted-foreground">Brutto-Gewinn</div>
                <div className="font-medium">{formatISKWithSeparators(route.gross_profit)}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Spread</div>
                <div className={cn("font-medium", getSpreadColor(route.spread_percent))}>
                  {route.spread_percent.toFixed(1)}%
                </div>
              </div>
            </div>

            {/* Fees Section with Tooltip */}
            {route.total_fees !== undefined && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="flex items-center justify-between cursor-help border-t pt-3">
                      <div className="flex items-center gap-1.5">
                        <span className="text-sm text-muted-foreground">Gebühren</span>
                        <Info className="size-3.5 text-muted-foreground" />
                      </div>
                      <span className="text-sm font-medium text-red-600 dark:text-red-400">
                        -{formatISKWithSeparators(route.total_fees)}
                      </span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent side="top" className="max-w-xs">
                    <FeeBreakdown
                      fees={{
                        salesTax: route.sales_tax || 0,
                        brokerFees: route.broker_fees || 0,
                        estimatedRelistFee: route.estimated_relist_fee || 0,
                        totalFees: route.total_fees,
                      }}
                    />
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}

            {/* Net Profit Section */}
            <div className="grid grid-cols-2 gap-4 border-t pt-3">
              <div>
                <div className="text-sm text-muted-foreground">Netto-Gewinn</div>
                <div className={cn("text-lg font-bold", getProfitColor(route.net_profit_percent || 0))}>
                  {formatISKWithSeparators(route.net_profit)}
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Netto-Marge</div>
                <div className={cn("text-lg font-bold", getProfitColor(route.net_profit_percent || 0))}>
                  {(route.net_profit_percent || 0).toFixed(1)}%
                </div>
              </div>
            </div>
          </div>
        ) : (
          /* Fallback to old display if fee data not available */
          <>
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
          </>
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
              : formatTime(route.round_trip_seconds)}
            {route.jumps !== undefined && (
              <span className="text-xs text-muted-foreground ml-1">
                ({route.jumps} {route.jumps === 1 ? 'Jump' : 'Jumps'})
              </span>
            )}
          </div>
          
          {/* Show skills improvement if available */}
          {route.time_improvement_percent != null && route.time_improvement_percent > 0 && (
            <div className="flex items-center gap-2 text-xs">
              {route.base_travel_time_seconds && route.base_travel_time_seconds > 0 && (
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
        {route.time_improvement_percent != null && route.time_improvement_percent > 5 && (
          <div className="border-t pt-3 flex items-center justify-between text-sm">
            <span className="text-muted-foreground">ISK/h Boost</span>
            <span className="inline-flex items-center gap-1 rounded-md bg-green-100 dark:bg-green-900/30 px-2 py-0.5 text-xs font-medium text-green-600 dark:text-green-400">
              <TrendingUp className="size-3" />
              +{route.time_improvement_percent.toFixed(0)}% durch Nav Skills
            </span>
          </div>
        )}

        {/* Cargo Utilization */}
        {route.cargo_capacity !== undefined && route.cargo_used !== undefined && (
          <div className="border-t pt-3">
            <div className="flex items-center justify-between text-sm mb-1.5">
              <span className="text-muted-foreground">Cargo</span>
              <span className="text-muted-foreground text-xs">
                {route.cargo_used.toFixed(0)} / {route.cargo_capacity.toFixed(0)} m³
                {route.skill_bonus_percent !== undefined && route.skill_bonus_percent > 0 && (
                  <span className="text-green-600 dark:text-green-400 ml-1">
                    (+{route.skill_bonus_percent.toFixed(1)}%)
                  </span>
                )}
              </span>
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
            {route.cargo_utilization !== undefined && route.cargo_utilization >= 95 && (
              <p className="text-xs text-green-600 dark:text-green-400 mt-1">
                ✅ Optimale Cargo-Auslastung ({route.cargo_utilization.toFixed(0)}%)
              </p>
            )}
          </div>
        )}

        {/* Volume & Liquidity Metrics */}
        {route.volume_metrics && (
          <div className="border-t pt-3">
            <div className="text-sm font-medium text-muted-foreground mb-2">
              Volume & Liquidität
            </div>
            
            {/* Daily Volume */}
            <div className="grid grid-cols-2 gap-4 mb-2 text-sm">
              <div>
                <div className="text-muted-foreground">Tägliches Volume</div>
                <div className="font-medium">
                  {route.volume_metrics.daily_volume_avg.toFixed(0)} Items
                </div>
              </div>
              <div>
                <div className="text-muted-foreground">ISK Umsatz/Tag</div>
                <div className="font-medium">
                  {formatISK(route.volume_metrics.daily_isk_turnover)}
                </div>
              </div>
            </div>

            {/* Liquidation Time */}
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

            {/* Daily Profit */}
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

            {/* Liquidity Score Bar */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Liquiditätsscore</span>
                <span className="font-medium">{route.volume_metrics.liquidity_score}/100</span>
              </div>
              <div className="h-2 bg-gray-700 dark:bg-gray-800 rounded-full overflow-hidden">
                <div
                  className={cn(
                    "h-full transition-all",
                    route.volume_metrics.liquidity_score >= 70
                      ? "bg-green-500"
                      : route.volume_metrics.liquidity_score >= 40
                      ? "bg-yellow-500"
                      : "bg-red-500"
                  )}
                  style={{ width: `${route.volume_metrics.liquidity_score}%` }}
                />
              </div>
            </div>

            {/* Data Quality Indicator */}
            {route.volume_metrics.data_days < 30 && (
              <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-2">
                ℹ️ Nur {route.volume_metrics.data_days} Tage Daten verfügbar
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

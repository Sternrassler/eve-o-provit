"use client";

import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { TradingRoute } from "@/types/trading";
import { cn } from "@/lib/utils";
import { useAuth } from "@/lib/auth-context";
import { useState } from "react";
import { useToast } from "@/hooks/use-toast";
import { RouteHeader } from "./RouteHeader";
import { RoutePathSection } from "./RoutePathSection";
import { RouteProfitSection } from "./RouteProfitSection";
import { RouteTravelSection } from "./RouteTravelSection";
import { RouteCargoSection } from "./RouteCargoSection";
import { RouteVolumeSection } from "./RouteVolumeSection";
import { setWaypoint } from "@/lib/api-client";

interface TradingRouteCardProps {
  route: TradingRoute;
}

function getSecurityBackground(route: TradingRoute): string {
  const minSecStatus =
    route.min_route_security_status ??
    Math.min(route.buy_security_status ?? 1.0, route.sell_security_status ?? 1.0);

  if (minSecStatus >= 0.5) return "bg-green-100/60 dark:bg-green-900/30";
  if (minSecStatus > 0.0) return "bg-yellow-100/60 dark:bg-yellow-900/30";
  return "bg-red-100/60 dark:bg-red-900/30";
}

export function TradingRouteCard({ route }: TradingRouteCardProps) {
  const { isAuthenticated } = useAuth();
  const { toast } = useToast();
  const [isSettingRoute, setIsSettingRoute] = useState(false);

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

    try {
      await navigator.clipboard.writeText(`${buyLink} → ${sellLink}`);
      toast({ title: "Link kopiert", description: "EVE Chat Link in Zwischenablage kopiert" });
    } catch {
      toast({ title: "Fehler", description: "Kopieren fehlgeschlagen", variant: "destructive" });
    }
  };

  const handleSetRoute = async () => {
    if (!isAuthenticated) {
      toast({ title: "Nicht eingeloggt", description: "EVE SSO Login erforderlich", variant: "destructive" });
      return;
    }
    if (!route.buy_station_id || !route.sell_station_id) {
      toast({ title: "Keine Stationen", description: "Route hat keine Station-IDs", variant: "destructive" });
      return;
    }

    setIsSettingRoute(true);
    try {
      await setWaypoint(route.buy_station_id, { clearOtherWaypoints: true });
      await setWaypoint(route.sell_station_id, { clearOtherWaypoints: false });

      toast({
        title: "Route gesetzt",
        description: `Waypoints in EVE gesetzt: ${route.buy_station_name} → ${route.sell_station_name}`,
      });
    } catch (err) {
      toast({
        title: "Fehler",
        description: err instanceof Error ? err.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    } finally {
      setIsSettingRoute(false);
    }
  };

  return (
    <Card
      className={cn(
        "transition-all hover:shadow-lg hover:border-primary/50",
        getSecurityBackground(route)
      )}
    >
      <CardHeader>
        <RouteHeader route={route} />
      </CardHeader>
      <CardContent className="space-y-3">
        <RoutePathSection
          route={route}
          isAuthenticated={isAuthenticated}
          isSettingRoute={isSettingRoute}
          onCopyLink={handleCopyLink}
          onSetRoute={handleSetRoute}
        />
        <RouteProfitSection route={route} />
        <RouteTravelSection route={route} />
        <RouteCargoSection route={route} />
        <RouteVolumeSection route={route} />
      </CardContent>
    </Card>
  );
}

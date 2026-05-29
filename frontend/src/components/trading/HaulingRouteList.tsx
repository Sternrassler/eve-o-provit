"use client";

import { useState } from "react";
import { ChevronDown, Loader2, Navigation } from "lucide-react";
import { HaulingRoute, HaulingItem, SecurityRisk } from "@/types/trading";
import { cn, formatISKWithSeparators } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { useAuth } from "@/lib/auth-context";
import { setWaypoint } from "@/lib/api-client";

interface HaulingRouteListProps {
  routes: HaulingRoute[];
}

const SECURITY_BADGE: Record<
  SecurityRisk,
  { label: string; className: string }
> = {
  safe: {
    label: "Sicher",
    className:
      "border-green-300 bg-green-50 text-green-700 dark:border-green-700 dark:bg-green-900/20 dark:text-green-300",
  },
  caution: {
    label: "Vorsicht",
    className:
      "border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300",
  },
  danger: {
    label: "Gefahr",
    className:
      "border-red-300 bg-red-50 text-red-700 dark:border-red-700 dark:bg-red-900/20 dark:text-red-300",
  },
};

function SecurityBadge({ risk }: { risk: SecurityRisk }) {
  const cfg = SECURITY_BADGE[risk] ?? SECURITY_BADGE.caution;
  return (
    <span
      data-testid="security-badge"
      data-risk={risk}
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold",
        cfg.className
      )}
    >
      {cfg.label}
    </span>
  );
}

/**
 * Profitable station→station hauling routes in the character's neighborhood.
 * Routes arrive pre-sorted by ISK/h (best first). Each route is expandable to a
 * cargo table and can be pushed to the in-game autopilot. An empty list means
 * no profitable routes were found nearby.
 */
export function HaulingRouteList({ routes }: HaulingRouteListProps) {
  if (routes.length === 0) {
    return (
      <div
        data-testid="hauling-empty"
        className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
      >
        Keine profitablen Routen im Umkreis
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="hauling-route-list">
      {routes.map((route, idx) => (
        <HaulingRouteCard
          key={`${route.buy_station_id}-${route.sell_station_id}-${idx}`}
          route={route}
        />
      ))}
    </div>
  );
}

function HaulingRouteCard({ route }: { route: HaulingRoute }) {
  const { isAuthenticated } = useAuth();
  const { toast } = useToast();
  const [expanded, setExpanded] = useState(false);
  const [isSettingRoute, setIsSettingRoute] = useState(false);

  const handleSetRoute = async () => {
    if (!isAuthenticated) {
      toast({
        title: "Nicht eingeloggt",
        description: "EVE SSO Login erforderlich",
        variant: "destructive",
      });
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
    <Card data-testid="hauling-route">
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle
              className="text-lg"
              title={`${route.buy_station_name} → ${route.sell_station_name}`}
            >
              {route.buy_system_name} → {route.sell_system_name}
            </CardTitle>
            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <SecurityBadge risk={route.security_risk} />
              <span>{route.jumps} Sprünge</span>
              <span aria-hidden>·</span>
              <span>{route.travel_time_min.toFixed(1)} Min</span>
              <span aria-hidden>·</span>
              <span>{route.total_volume.toLocaleString("de-DE")} m³</span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleSetRoute}
              disabled={isSettingRoute}
              title="Route an EVE übertragen"
              aria-label="Route an EVE übertragen"
            >
              {isSettingRoute ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Navigation className="size-4" />
              )}
              <span className="ml-2 hidden sm:inline">
                Route an EVE übertragen
              </span>
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Metric label="ISK/h" value={formatISKWithSeparators(route.isk_per_hour)} positive />
          <Metric
            label="Gesamtgewinn"
            value={formatISKWithSeparators(route.total_profit)}
            positive
          />
          <Metric
            label="Kapitaleinsatz"
            value={formatISKWithSeparators(route.total_capital)}
          />
          <Metric
            label="Volumen"
            value={`${route.total_volume.toLocaleString("de-DE")} m³`}
          />
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          className="w-full justify-between"
          data-testid="hauling-expand"
        >
          <span>Ladung ({route.items.length})</span>
          <ChevronDown
            className={cn(
              "size-4 transition-transform",
              expanded && "rotate-180"
            )}
          />
        </Button>

        {expanded && <CargoTable items={route.items} route={route} />}
      </CardContent>
    </Card>
  );
}

function Metric({
  label,
  value,
  positive,
}: {
  label: string;
  value: string;
  positive?: boolean;
}) {
  return (
    <div className="space-y-0.5">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className={cn(
          "font-semibold",
          positive && "text-green-600 dark:text-green-400"
        )}
      >
        {value}
      </div>
    </div>
  );
}

function CargoTable({
  items,
  route,
}: {
  items: HaulingItem[];
  route: HaulingRoute;
}) {
  return (
    <div className="overflow-x-auto" data-testid="cargo-table">
      <table className="w-full text-sm" aria-label="Ladung">
        <thead>
          <tr className="border-b text-left text-muted-foreground">
            <th scope="col" className="px-3 py-2 font-medium">
              Item
            </th>
            <th scope="col" className="px-3 py-2 text-right font-medium">
              Menge
            </th>
            <th scope="col" className="px-3 py-2 text-right font-medium">
              Kauf@{route.buy_system_name}
            </th>
            <th scope="col" className="px-3 py-2 text-right font-medium">
              Verkauf@{route.sell_system_name}
            </th>
            <th scope="col" className="px-3 py-2 text-right font-medium">
              Volumen
            </th>
            <th scope="col" className="px-3 py-2 text-right font-medium">
              Gewinn
            </th>
            <th scope="col" className="px-3 py-2 text-right font-medium">
              Gewinn/m³
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={item.type_id}
              data-testid="cargo-row"
              data-type-id={item.type_id}
              className="border-b transition-colors hover:bg-muted/40"
            >
              <td className="px-3 py-2 font-medium">{item.name}</td>
              <td className="px-3 py-2 text-right">
                {item.quantity.toLocaleString("de-DE")}
              </td>
              <td className="px-3 py-2 text-right">
                {formatISKWithSeparators(item.buy_price)}
              </td>
              <td className="px-3 py-2 text-right">
                {formatISKWithSeparators(item.sell_price)}
              </td>
              <td className="px-3 py-2 text-right">
                {(item.unit_volume * item.quantity).toLocaleString("de-DE")} m³
              </td>
              <td className="px-3 py-2 text-right font-medium text-green-600 dark:text-green-400">
                {formatISKWithSeparators(item.profit)}
              </td>
              <td className="px-3 py-2 text-right">
                {formatISKWithSeparators(item.profit_per_m3)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

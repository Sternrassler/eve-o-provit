"use client";

import { useState } from "react";
import { Loader2, Navigation, Trophy } from "lucide-react";
import { SellOption, SellOptionScope, SecurityRisk } from "@/types/trading";
import { cn, formatISKWithSeparators } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { useAuth } from "@/lib/auth-context";
import { setWaypoint } from "@/lib/api-client";

interface SellOptionsResultProps {
  best: SellOption | null;
  options: SellOption[];
  /** Set when the empty result has a known cause; UI shows an actionable hint
   *  instead of the generic "no buyers" message. */
  notRoutableReason?: string;
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

function ScopeBadge({ scope }: { scope: SellOptionScope }) {
  return (
    <Badge variant="outline" className="text-xs" data-testid="scope-badge">
      {scope === "hub" ? "Hub" : "Region"}
    </Badge>
  );
}

/**
 * Ranked sell locations for an owned item (taker model, net of sales tax). The
 * best actionable option is highlighted at the top, followed by the full list
 * pre-sorted by ISK/h (local sales first, then remote by net-per-hour). Each
 * option can be pushed to the in-game autopilot.
 * Options without market data render muted. An empty list shows a hint.
 */
export function SellOptionsResult({
  best,
  options,
  notRoutableReason,
}: SellOptionsResultProps) {
  // Tolerate `null` from older backends / unresolvable origins (player structures
  // return an empty path that historically marshaled as JSON null).
  const safeOptions = options ?? [];
  if (safeOptions.length === 0) {
    const isCitadelOrigin = notRoutableReason === "origin_in_player_structure";
    return (
      <div
        data-testid="sell-options-empty"
        data-reason={notRoutableReason || undefined}
        className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
      >
        {isCitadelOrigin ? (
          <>
            <p className="font-medium">
              Items liegen in einer Player-Structure (Citadel/Upwell).
            </p>
            <p className="mt-1">
              Routen können wir aus Player-Structures nicht berechnen — die
              SDE kennt nur NPC-Stationen. Verlagere den Bestand in eine
              NPC-Station; dann erscheinen hier die Verkaufsorte. (ROI/Trading
              zeigen die Marktdaten weiterhin an, weil sie regionsweit
              rechnen, nicht von deinem Standort.)
            </p>
          </>
        ) : (
          "Keine Verkaufsorte gefunden"
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="sell-options-result">
      {best && (
        <SellOptionCard
          key={`best-${best.station_id}`}
          option={best}
          highlighted
        />
      )}
      <div className="space-y-3" data-testid="sell-options-list">
        {safeOptions.map((option, idx) => (
          <SellOptionCard
            key={`${option.station_id}-${idx}`}
            option={option}
          />
        ))}
      </div>
    </div>
  );
}

function SellOptionCard({
  option,
  highlighted,
}: {
  option: SellOption;
  highlighted?: boolean;
}) {
  const { isAuthenticated } = useAuth();
  const { toast } = useToast();
  const [isSettingRoute, setIsSettingRoute] = useState(false);

  const handleSetWaypoint = async () => {
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
      await setWaypoint(option.station_id, { clearOtherWaypoints: true });
      toast({
        title: "Route gesetzt",
        description: `Waypoint in EVE gesetzt: ${option.station_name}`,
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
      data-testid="sell-option"
      data-has-data={option.has_data}
      data-highlighted={highlighted ? "true" : undefined}
      className={cn(
        highlighted && "border-primary ring-1 ring-primary",
        !option.has_data && "opacity-60"
      )}
    >
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle
              className="flex items-center gap-2 text-lg"
              title={option.station_name}
            >
              {highlighted && (
                <Trophy
                  className="size-4 text-amber-500"
                  aria-label="Beste Option"
                />
              )}
              {option.system_name}
            </CardTitle>
            <div className="text-sm text-muted-foreground">
              {option.station_name}
            </div>
            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <ScopeBadge scope={option.scope} />
              <SecurityBadge risk={option.security_risk} />
              <span>{option.jumps} Sprünge</span>
              <span aria-hidden>·</span>
              <span>{option.travel_time_min.toFixed(1)} Min</span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleSetWaypoint}
              disabled={isSettingRoute || !option.has_data}
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
      <CardContent>
        {option.has_data ? (
          <div className="grid grid-cols-4 gap-4">
            <Metric
              label="Kaufpreis"
              value={formatISKWithSeparators(option.buy_price)}
            />
            <Metric
              label="Netto/Stück"
              value={formatISKWithSeparators(option.unit_net)}
            />
            <Metric
              label="Netto gesamt"
              value={formatISKWithSeparators(option.total_net)}
              positive
            />
            <Metric
              label="ISK/h"
              value={
                option.travel_time_min > 0
                  ? formatISKWithSeparators(option.isk_per_hour)
                  : "sofort"
              }
              positive
            />
          </div>
        ) : (
          <div
            data-testid="sell-option-no-data"
            className="text-sm text-muted-foreground"
          >
            kein Marktpreis
          </div>
        )}
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

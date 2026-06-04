"use client";

import { useState, useMemo } from "react";
import { TradingTabs } from "@/components/trading/TradingTabs";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { RegionSelect } from "@/components/trading/RegionSelect";
import { CurrentShipCard } from "@/components/trading/CurrentShipCard";
import { ShipFittingCard } from "@/components/trading/ShipFittingCard";
import { TradingRouteList } from "@/components/trading/TradingRouteList";
import { TradingFilters } from "@/components/trading/TradingFilters";
import { Button } from "@/components/ui/button";
import { TradingFilters as TradingFiltersType, TradingRoute } from "@/types/trading";
import { fetchCharacterLocation } from "@/lib/api-client";
import { useCurrentShip } from "@/lib/use-current-ship";
import { Loader2 } from "lucide-react";

const MAX_DISPLAYED_ROUTES = 50;
const DEFAULT_REGION = "10000002"; // The Forge
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

const defaultFilters: TradingFiltersType = {
  minSpread: 5,
  minProfit: 100000,
  maxTravelTime: 30,
  allowHighSec: true,
  allowLowSec: false,
  allowNullSec: false,
};

async function calculateRoutes(
  regionId: number,
  shipTypeId: number,
  cargoCapacity?: number
): Promise<TradingRoute[]> {
  const response = await fetch(`${API_BASE_URL}/api/v1/trading/routes/calculate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({
      region_id: regionId,
      ship_type_id: shipTypeId,
      ...(cargoCapacity && cargoCapacity > 0 ? { cargo_capacity: cargoCapacity } : {}),
    }),
  });

  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data = await response.json();
  if (!Array.isArray(data.routes)) {
    throw new Error("Invalid routes response: 'routes' missing or not an array");
  }
  return data.routes;
}

function TradingPageContent() {
  const { isAuthenticated } = useAuth();
  // null = keine manuelle Auswahl; effektiver Wert wird unten aus Character-Daten abgeleitet
  const [regionOverride, setRegionOverride] = useState<string | null>(null);
  const [filters, setFilters] = useState<TradingFiltersType>(defaultFilters);
  const [displayedRoutes, setDisplayedRoutes] = useState(10);
  const [isRefreshingMarketData, setIsRefreshingMarketData] = useState(false);

  // Das aktuelle Schiff steuert ship_type_id + Cargo-Override (kein Dropdown mehr).
  const { ship: currentShip } = useCurrentShip();

  // Load character location when authenticated (für Region-Prefill + ShipFittingCard).
  const { data: characterData, isPending: characterDataLoading } = useQuery({
    queryKey: ["characterData", isAuthenticated],
    queryFn: async () => {
      const location = await fetchCharacterLocation();
      return { location };
    },
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });

  // Effektive Region: manuelle Override > Character-Daten > Default. Rein abgeleitet,
  // kein Effect/Ref — die User-Auswahl (Override) bleibt sticky.
  const selectedRegion =
    regionOverride ?? characterData?.location.region_id?.toString() ?? DEFAULT_REGION;

  const characterId = characterData?.location.character_id ?? null;

  // Cargo-Override aus dem aktuellen Schiff: nur die effektive (fitted) Kapazität,
  // wenn verfügbar und > 0 — sonst überlässt das Backend dem ship_type_id die Basis.
  const cargoOverride =
    currentShip &&
    !currentShip.effective_cargo_unavailable &&
    currentShip.effective_cargo_capacity != null &&
    currentShip.effective_cargo_capacity > 0
      ? currentShip.effective_cargo_capacity
      : undefined;

  // Route calculation mutation (triggered by button click)
  const routeMutation = useMutation({
    mutationFn: () =>
      calculateRoutes(parseInt(selectedRegion), currentShip!.ship_type_id, cargoOverride),
    onSuccess: () => setDisplayedRoutes(10),
  });

  const filteredRoutes = useMemo(() => {
    if (!routeMutation.isSuccess || !routeMutation.data) return [];

    return routeMutation.data.filter((route) => {
      const travelTimeMinutes = route.travel_time_seconds / 60;
      const totalProfit = route.total_profit ?? route.profit ?? 0;
      const netProfit = route.net_profit ?? totalProfit;

      if (netProfit < 0) return false;
      if (route.spread_percent < filters.minSpread) return false;
      if (totalProfit < filters.minProfit) return false;
      if (travelTimeMinutes > filters.maxTravelTime) return false;

      const minSecStatus =
        route.min_route_security_status ??
        Math.min(route.buy_security_status ?? 1.0, route.sell_security_status ?? 1.0);

      const isHighSec = minSecStatus >= 0.5;
      const isLowSec = minSecStatus > 0.0 && minSecStatus < 0.5;
      const isNullSec = minSecStatus <= 0.0;

      if (isHighSec && !filters.allowHighSec) return false;
      if (isLowSec && !filters.allowLowSec) return false;
      if (isNullSec && !filters.allowNullSec) return false;

      return true;
    });
  }, [routeMutation.isSuccess, routeMutation.data, filters]);

  const visibleRoutes = filteredRoutes.slice(0, displayedRoutes);
  const hasMoreRoutes =
    displayedRoutes < filteredRoutes.length &&
    displayedRoutes < MAX_DISPLAYED_ROUTES;

  const isCalculateDisabled =
    !selectedRegion ||
    !currentShip ||
    routeMutation.isPending ||
    characterDataLoading ||
    isRefreshingMarketData;

  const apiError = routeMutation.isError
    ? routeMutation.error instanceof Error
      ? routeMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  return (
    <div className="container mx-auto px-4 py-8">
      <TradingTabs />
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Trading</h1>
        <p className="text-muted-foreground">
          Optimiere deine Handelsrouten für maximalen Profit
        </p>
      </div>

      <div className="mb-8 grid gap-6 lg:grid-cols-[300px_1fr]">
        {/* Sidebar */}
        <div className="space-y-6">
          <div className="space-y-4 rounded-lg border p-4">
            <RegionSelect
              value={selectedRegion}
              onChange={setRegionOverride}
              disabled={routeMutation.isPending || characterDataLoading}
              onRefreshStateChange={setIsRefreshingMarketData}
            />
            <CurrentShipCard />
            <Button
              className="w-full"
              onClick={() => routeMutation.mutate()}
              disabled={isCalculateDisabled}
            >
              {(routeMutation.isPending || characterDataLoading) && (
                <Loader2 className="mr-2 size-4 animate-spin" />
              )}
              {characterDataLoading
                ? "Lade Character-Daten..."
                : routeMutation.isPending
                ? "Berechne..."
                : "Berechnen"}
            </Button>
          </div>

          {isAuthenticated && characterId && currentShip && (
            <ShipFittingCard
              characterId={characterId}
              shipTypeId={currentShip.ship_type_id}
            />
          )}

          <TradingFilters filters={filters} onChange={setFilters} />
        </div>

        {/* Results */}
        <div className="space-y-6">
          {routeMutation.isSuccess && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {filteredRoutes.length} Routen gefunden
                {filteredRoutes.length > displayedRoutes &&
                  ` (${displayedRoutes} angezeigt)`}
              </p>
            </div>
          )}

          <TradingRouteList
            routes={visibleRoutes}
            loading={routeMutation.isPending}
            error={apiError}
            onRetry={() => routeMutation.mutate()}
          />

          {routeMutation.isSuccess && hasMoreRoutes && (
            <div className="flex justify-center">
              <Button
                variant="outline"
                onClick={() =>
                  setDisplayedRoutes((prev) =>
                    Math.min(prev + 10, MAX_DISPLAYED_ROUTES)
                  )
                }
              >
                Mehr anzeigen (noch{" "}
                {Math.min(filteredRoutes.length - displayedRoutes, 10)})
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function TradingPage() {
  return (
    <ErrorBoundary>
      <TradingPageContent />
    </ErrorBoundary>
  );
}

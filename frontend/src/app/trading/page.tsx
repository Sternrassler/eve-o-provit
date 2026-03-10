"use client";

import { useState, useMemo, useEffect } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { RegionSelect } from "@/components/trading/RegionSelect";
import { ShipSelect } from "@/components/trading/ShipSelect";
import { ShipFittingCard } from "@/components/trading/ShipFittingCard";
import { TradingRouteList } from "@/components/trading/TradingRouteList";
import { TradingFilters } from "@/components/trading/TradingFilters";
import { Button } from "@/components/ui/button";
import { TradingFilters as TradingFiltersType, TradingRoute } from "@/types/trading";
import { fetchCharacterLocation, fetchCharacterShip } from "@/lib/api-client";
import { Loader2 } from "lucide-react";

const MAX_DISPLAYED_ROUTES = 50;
const DEFAULT_REGION = "10000002"; // The Forge
const DEFAULT_SHIP = "648";
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

const defaultFilters: TradingFiltersType = {
  minSpread: 5,
  minProfit: 100000,
  maxTravelTime: 30,
  allowHighSec: true,
  allowLowSec: false,
  allowNullSec: false,
};

async function calculateRoutes(regionId: number, shipTypeId: number): Promise<TradingRoute[]> {
  const response = await fetch(`${API_BASE_URL}/api/v1/trading/routes/calculate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ region_id: regionId, ship_type_id: shipTypeId }),
  });

  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data = await response.json();
  return data.routes || [];
}

function TradingPageContent() {
  const { isAuthenticated } = useAuth();
  const [selectedRegion, setSelectedRegion] = useState<string>(DEFAULT_REGION);
  const [selectedShip, setSelectedShip] = useState<string>(DEFAULT_SHIP);
  const [filters, setFilters] = useState<TradingFiltersType>(defaultFilters);
  const [displayedRoutes, setDisplayedRoutes] = useState(10);
  const [isRefreshingMarketData, setIsRefreshingMarketData] = useState(false);

  // Load character location + ship when authenticated
  const { data: characterData, isPending: characterDataLoading } = useQuery({
    queryKey: ["characterData", isAuthenticated],
    queryFn: async () => {
      const [location, ship] = await Promise.all([
        fetchCharacterLocation(),
        fetchCharacterShip(),
      ]);
      return { location, ship };
    },
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });

  // Apply character data to selections
  useEffect(() => {
    if (!characterData) return;
    if (characterData.location.region_id) {
      setSelectedRegion(characterData.location.region_id.toString());
    }
    if (characterData.ship.ship_type_id) {
      setSelectedShip(characterData.ship.ship_type_id.toString());
    }
  }, [characterData]);

  const characterId = characterData?.location.character_id ?? null;

  // Route calculation mutation (triggered by button click)
  const routeMutation = useMutation({
    mutationFn: () =>
      calculateRoutes(parseInt(selectedRegion), parseInt(selectedShip)),
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
    !selectedShip ||
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
              onChange={setSelectedRegion}
              disabled={routeMutation.isPending || characterDataLoading}
              onRefreshStateChange={setIsRefreshingMarketData}
            />
            <ShipSelect
              value={selectedShip}
              onChange={setSelectedShip}
              disabled={routeMutation.isPending || characterDataLoading}
              authenticated={isAuthenticated}
            />
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

          {isAuthenticated && characterId && selectedShip && (
            <ShipFittingCard
              characterId={characterId}
              shipTypeId={parseInt(selectedShip)}
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

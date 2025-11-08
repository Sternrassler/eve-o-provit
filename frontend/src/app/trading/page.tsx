"use client";

import { useState, useMemo, useEffect } from "react";
import { useAuth } from "@/lib/auth-context";
import { RegionSelect } from "@/components/trading/RegionSelect";
import { ShipSelect } from "@/components/trading/ShipSelect";
import { TradingRouteList } from "@/components/trading/TradingRouteList";
import { TradingFilters } from "@/components/trading/TradingFilters";
import { Button } from "@/components/ui/button";
import { TradingFilters as TradingFiltersType, TradingRoute } from "@/types/trading";
import { fetchCharacterLocation, fetchCharacterShip } from "@/lib/api-client";
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

export default function TradingPage() {
  const { isAuthenticated, getAuthHeader } = useAuth();
  const [selectedRegion, setSelectedRegion] = useState<string>(DEFAULT_REGION);
  const [selectedShip, setSelectedShip] = useState<string>("648");
  const [filters, setFilters] = useState<TradingFiltersType>(defaultFilters);
  const [isCalculating, setIsCalculating] = useState(false);
  const [hasCalculated, setHasCalculated] = useState(false);
  const [displayedRoutes, setDisplayedRoutes] = useState(10);
  const [apiRoutes, setApiRoutes] = useState<TradingRoute[]>([]);
  const [apiError, setApiError] = useState<string | null>(null);
  const [characterDataLoading, setCharacterDataLoading] = useState(false);
  const [isRefreshingMarketData, setIsRefreshingMarketData] = useState(false);

  // Load character data when authenticated
  useEffect(() => {
    const loadCharacterData = async () => {
      if (!isAuthenticated) return;

      const authHeader = getAuthHeader();
      if (!authHeader) return;

      setCharacterDataLoading(true);
      
      try {
        // Fetch character location to get region
        const location = await fetchCharacterLocation(authHeader);
        if (location.region_id) {
          setSelectedRegion(location.region_id.toString());
        }

        // Fetch current ship
        const ship = await fetchCharacterShip(authHeader);
        if (ship.ship_type_id) {
          setSelectedShip(ship.ship_type_id.toString());
        }
      } catch (error) {
        console.error("Failed to load character data:", error);
        // Keep default values on error
      } finally {
        setCharacterDataLoading(false);
      }
    };

    loadCharacterData();
  }, [isAuthenticated, getAuthHeader]);

  const handleCalculate = async () => {
    setIsCalculating(true);
    setHasCalculated(false);
    setApiError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/trading/routes/calculate`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          region_id: parseInt(selectedRegion),
          ship_type_id: parseInt(selectedShip),
        }),
      });

      if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
      }

      const data = await response.json();
      setApiRoutes(data.routes || []);
      setHasCalculated(true);
    } catch (error) {
      console.error("Failed to calculate routes:", error);
      setApiError(error instanceof Error ? error.message : "Unknown error");
      setApiRoutes([]);
    } finally {
      setIsCalculating(false);
      setDisplayedRoutes(10);
    }
  };

  const filteredRoutes = useMemo(() => {
    if (!hasCalculated) return [];

    return apiRoutes.filter((route) => {
      const travelTimeMinutes = route.travel_time_seconds / 60;
      const totalProfit = route.total_profit ?? route.profit ?? 0;
      
      // Filter out routes with negative net profit (loss)
      const netProfit = route.net_profit ?? totalProfit;
      if (netProfit < 0) return false;
      
      // Check basic filters
      if (route.spread_percent < filters.minSpread) return false;
      if (totalProfit < filters.minProfit) return false;
      if (travelTimeMinutes > filters.maxTravelTime) return false;

      // Check security zone filters
      // Use min_route_security_status if available (considers entire route)
      // Otherwise fall back to min of buy/sell (backward compatibility)
      const minSecStatus = route.min_route_security_status ?? 
                           Math.min(route.buy_security_status ?? 1.0, route.sell_security_status ?? 1.0);

      const isHighSec = minSecStatus >= 0.5;
      const isLowSec = minSecStatus > 0.0 && minSecStatus < 0.5;
      const isNullSec = minSecStatus <= 0.0;

      if (isHighSec && !filters.allowHighSec) return false;
      if (isLowSec && !filters.allowLowSec) return false;
      if (isNullSec && !filters.allowNullSec) return false;

      return true;
    });
  }, [hasCalculated, apiRoutes, filters]);

  const visibleRoutes = filteredRoutes.slice(0, displayedRoutes);
  const hasMoreRoutes = displayedRoutes < filteredRoutes.length && displayedRoutes < MAX_DISPLAYED_ROUTES;

  const handleShowMore = () => {
    setDisplayedRoutes((prev) => Math.min(prev + 10, MAX_DISPLAYED_ROUTES));
  };

  const isCalculateDisabled = !selectedRegion || !selectedShip || isCalculating || characterDataLoading || isRefreshingMarketData;

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Trading</h1>
        <p className="text-muted-foreground">
          Optimiere deine Handelsrouten für maximalen Profit
        </p>
      </div>

      {/* Control Panel */}
      <div className="mb-8 grid gap-6 lg:grid-cols-[300px_1fr]">
        {/* Sidebar */}
        <div className="space-y-6">
          {/* Region & Ship Selection */}
          <div className="space-y-4 rounded-lg border p-4">
            <RegionSelect
              value={selectedRegion}
              onChange={setSelectedRegion}
              disabled={isCalculating || characterDataLoading}
              onRefreshStateChange={setIsRefreshingMarketData}
            />
            <ShipSelect
              value={selectedShip}
              onChange={setSelectedShip}
              disabled={isCalculating || characterDataLoading}
              authenticated={isAuthenticated}
              authHeader={getAuthHeader()}
            />
            <Button
              className="w-full"
              onClick={handleCalculate}
              disabled={isCalculateDisabled}
            >
              {(isCalculating || characterDataLoading) && <Loader2 className="mr-2 size-4 animate-spin" />}
              {characterDataLoading ? "Lade Character-Daten..." : isCalculating ? "Berechne..." : "Berechnen"}
            </Button>
          </div>

          {/* Filters */}
          <TradingFilters filters={filters} onChange={setFilters} />
        </div>

        {/* Results Section */}
        <div className="space-y-6">
          {hasCalculated && (
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
            loading={isCalculating}
            error={apiError || undefined}
            onRetry={handleCalculate}
          />

          {hasCalculated && hasMoreRoutes && (
            <div className="flex justify-center">
              <Button variant="outline" onClick={handleShowMore}>
                Mehr anzeigen (noch {Math.min(filteredRoutes.length - displayedRoutes, 10)})
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

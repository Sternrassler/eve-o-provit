"use client";

import { useState, useMemo } from "react";
import { RegionSelect } from "@/components/trading/RegionSelect";
import { ShipSelect } from "@/components/trading/ShipSelect";
import { TradingRouteList } from "@/components/trading/TradingRouteList";
import { TradingFilters } from "@/components/trading/TradingFilters";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { TradingFilters as TradingFiltersType } from "@/types/trading";
import { mockTradingRoutes } from "@/lib/mock-data/trading-routes";
import { Loader2 } from "lucide-react";

const MAX_DISPLAYED_ROUTES = 50;

const defaultFilters: TradingFiltersType = {
  minSpread: 5,
  minProfit: 100000,
  maxTravelTime: 30,
};

export default function IntraRegionPage() {
  const [selectedRegion, setSelectedRegion] = useState<string>("10000002");
  const [selectedShip, setSelectedShip] = useState<string>("648");
  const [filters, setFilters] = useState<TradingFiltersType>(defaultFilters);
  const [isCalculating, setIsCalculating] = useState(false);
  const [hasCalculated, setHasCalculated] = useState(false);
  const [displayedRoutes, setDisplayedRoutes] = useState(10);
  const [sortBy, setSortBy] = useState<
    "isk_per_hour" | "profit" | "spread_percent" | "travel_time_seconds"
  >("isk_per_hour");
  const [apiRoutes, setApiRoutes] = useState<any[]>([]);
  const [apiError, setApiError] = useState<string | null>(null);

  const handleCalculate = async () => {
    setIsCalculating(true);
    setHasCalculated(false);
    setApiError(null);

    try {
      const response = await fetch("http://localhost:9001/api/v1/trading/routes/calculate", {
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

    const routes = apiRoutes.filter((route) => {
      const travelTimeMinutes = route.travel_time_seconds / 60;
      return (
        route.spread_percent >= filters.minSpread &&
        route.total_profit >= filters.minProfit &&
        travelTimeMinutes <= filters.maxTravelTime
      );
    });

    routes.sort((a, b) => {
      switch (sortBy) {
        case "isk_per_hour":
          return b.isk_per_hour - a.isk_per_hour;
        case "profit":
          return b.total_profit - a.total_profit;
        case "spread_percent":
          return b.spread_percent - a.spread_percent;
        case "travel_time_seconds":
          return a.travel_time_seconds - b.travel_time_seconds;
        default:
          return 0;
      }
    });

    return routes;
  }, [hasCalculated, apiRoutes, filters, sortBy]);

  const visibleRoutes = filteredRoutes.slice(0, displayedRoutes);
  const hasMoreRoutes = displayedRoutes < filteredRoutes.length && displayedRoutes < MAX_DISPLAYED_ROUTES;

  const handleShowMore = () => {
    setDisplayedRoutes((prev) => Math.min(prev + 10, MAX_DISPLAYED_ROUTES));
  };

  const isCalculateDisabled = !selectedRegion || !selectedShip || isCalculating;

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-4xl font-bold">Intra-Region Trading</h1>
        <p className="mt-2 text-muted-foreground">
          Optimiere deine Handelsrouten innerhalb einer Region
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
              disabled={isCalculating}
            />
            <ShipSelect
              value={selectedShip}
              onChange={setSelectedShip}
              disabled={isCalculating}
              authenticated={false}
            />
            <Button
              className="w-full"
              onClick={handleCalculate}
              disabled={isCalculateDisabled}
            >
              {isCalculating && <Loader2 className="mr-2 size-4 animate-spin" />}
              {isCalculating ? "Berechne..." : "Berechnen"}
            </Button>
          </div>

          {/* Filters */}
          {hasCalculated && (
            <TradingFilters filters={filters} onChange={setFilters} />
          )}

          {/* Sort Options */}
          {hasCalculated && filteredRoutes.length > 0 && (
            <div className="space-y-2 rounded-lg border p-4">
              <label className="text-sm font-medium">Sortieren nach</label>
              <Select value={sortBy} onValueChange={(value) => setSortBy(value as typeof sortBy)}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="isk_per_hour">ISK/Hour</SelectItem>
                  <SelectItem value="profit">Gewinn</SelectItem>
                  <SelectItem value="spread_percent">Spread</SelectItem>
                  <SelectItem value="travel_time_seconds">Reisezeit</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
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

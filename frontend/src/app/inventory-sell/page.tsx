"use client";

import { useState, useEffect } from "react";
import { useAuth } from "@/lib/auth-context";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ItemAutocomplete } from "@/components/item-autocomplete";
import { ItemSearchResult, InventorySellRequest, InventorySellRoute } from "@/types/trading";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { RegionStalenessIndicator } from "@/components/trading/RegionStalenessIndicator";
import { RegionRefreshButton } from "@/components/trading/RegionRefreshButton";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

interface Region {
  id: number;
  name: string;
}

export default function InventorySellPage() {
  const { isAuthenticated, character, getAuthHeader } = useAuth();
  const [selectedItem, setSelectedItem] = useState<ItemSearchResult | null>(null);
  const [quantity, setQuantity] = useState("");
  const [buyPrice, setBuyPrice] = useState("");
  const [regionId, setRegionId] = useState("10000002");
  const [minProfit, setMinProfit] = useState("0");
  const [securityFilter, setSecurityFilter] = useState("all");
  const [routes, setRoutes] = useState<InventorySellRoute[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [regions, setRegions] = useState<Region[]>([]);
  const [isLoadingRegions, setIsLoadingRegions] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  // Load regions and character location on mount
  useEffect(() => {
    const loadRegionsAndLocation = async () => {
      try {
        // Load all regions from SDE
        const regionsResponse = await fetch(`${API_BASE_URL}/api/v1/sde/regions`);
        if (regionsResponse.ok) {
          const data = await regionsResponse.json();
          setRegions(data.regions || []);
        }

        // If authenticated, get character location and set region
        if (isAuthenticated && character) {
          const authHeader = getAuthHeader();
          if (authHeader) {
            const locationResponse = await fetch(`${API_BASE_URL}/api/v1/character/location`, {
              headers: {
                Authorization: authHeader,
              },
            });
            if (locationResponse.ok) {
              const locationData = await locationResponse.json();
              if (locationData.region_id) {
                setRegionId(locationData.region_id.toString());
              }
            }
          }
        }
      } catch (err) {
        console.error("Failed to load regions or location:", err);
      } finally {
        setIsLoadingRegions(false);
      }
    };

    loadRegionsAndLocation();
  }, [isAuthenticated, character, getAuthHeader]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedItem) {
      setError("Bitte ein Item auswählen");
      return;
    }
    if (!isAuthenticated || !character) {
      setError("Nicht eingeloggt");
      return;
    }

    const authHeader = getAuthHeader();
    if (!authHeader) {
      setError("Keine Authentifizierung verfügbar");
      return;
    }

    setError("");
    setIsLoading(true);

    try {
      const request: InventorySellRequest = {
        type_id: selectedItem.type_id,
        quantity: parseInt(quantity) || 1,
        buy_price_per_unit: parseFloat(buyPrice) || 0,
        region_id: parseInt(regionId),
        min_profit_per_unit: parseFloat(minProfit) || 0,
        security_filter: securityFilter,
      };

      const response = await fetch(`${API_BASE_URL}/api/v1/trading/inventory-sell`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: authHeader,
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Fehler beim Berechnen der Routen");
      }

      const data = await response.json();
      console.log("API Response:", data);
      console.log("Routes count:", data.routes?.length || 0);
      setRoutes(data.routes || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ein Fehler ist aufgetreten");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSetRoute = async (route: InventorySellRoute) => {
    if (!isAuthenticated || !character) return;

    const authHeader = getAuthHeader();
    if (!authHeader) return;

    try {
      for (const systemId of route.route_system_ids) {
        await fetch(`${API_BASE_URL}/api/v1/waypoint/set`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: authHeader,
          },
          body: JSON.stringify({
            destination_id: systemId,
            add_to_beginning: false,
            clear_other_waypoints: false,
          }),
        });
      }
    } catch (err) {
      console.error("Failed to set route:", err);
    }
  };

  const getSecurityColor = (security: number) => {
    if (security >= 0.5) return "text-green-600";
    if (security > 0.0) return "text-yellow-600";
    return "text-red-600";
  };

  const handleRefreshComplete = () => {
    setRefreshKey((prev) => prev + 1);
  };

  if (!isAuthenticated || !character) {
    return (
      <div className="container mx-auto p-6">
        <h1 className="mb-4 text-2xl font-bold">Inventory Sell</h1>
        <p>Bitte einloggen um diese Funktion zu nutzen.</p>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <h1 className="mb-6 text-3xl font-bold">Inventory Sell</h1>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Item & Parameter</CardTitle>
          <CardDescription>Wähle ein Item und gib die Verkaufsparameter ein</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="item" className="block text-sm font-medium mb-2">Item</label>
              <ItemAutocomplete
                value={selectedItem}
                onChange={setSelectedItem}
                apiUrl={API_BASE_URL}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="quantity" className="block text-sm font-medium mb-2">Menge</label>
                <Input
                  id="quantity"
                  type="number"
                  min="1"
                  value={quantity}
                  onChange={(e) => setQuantity(e.target.value)}
                  placeholder="1"
                />
              </div>

              <div>
                <label htmlFor="buyPrice" className="block text-sm font-medium mb-2">Einkaufspreis (pro Stück)</label>
                <Input
                  id="buyPrice"
                  type="number"
                  step="0.01"
                  min="0"
                  value={buyPrice}
                  onChange={(e) => setBuyPrice(e.target.value)}
                  placeholder="0.00"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="region" className="block text-sm font-medium mb-2">Region</label>
                <div className="flex items-center gap-2">
                  <Select value={regionId} onValueChange={setRegionId} disabled={isLoadingRegions}>
                    <SelectTrigger id="region">
                      <SelectValue placeholder={isLoadingRegions ? "Lade Regionen..." : "Region wählen"} />
                    </SelectTrigger>
                    <SelectContent>
                      {regions.map((region) => (
                        <SelectItem key={region.id} value={region.id.toString()}>
                          {region.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <RegionRefreshButton 
                    regionId={regionId}
                    disabled={isLoadingRegions || !regionId}
                    onRefreshComplete={handleRefreshComplete}
                  />
                </div>
                {regionId && (
                  <RegionStalenessIndicator 
                    key={refreshKey}
                    regionId={regionId}
                    className="mt-2"
                  />
                )}
              </div>

              <div>
                <label htmlFor="minProfit" className="block text-sm font-medium mb-2">Min. Profit (pro Stück)</label>
                <Input
                  id="minProfit"
                  type="number"
                  step="0.01"
                  min="0"
                  value={minProfit}
                  onChange={(e) => setMinProfit(e.target.value)}
                  placeholder="0.00"
                />
              </div>
            </div>

            <div>
              <label htmlFor="security" className="block text-sm font-medium mb-2">Sicherheitsfilter</label>
              <Select value={securityFilter} onValueChange={setSecurityFilter}>
                <SelectTrigger id="security">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Alle</SelectItem>
                  <SelectItem value="highsec">Nur High-Sec (≥0.5)</SelectItem>
                  <SelectItem value="highlow">High + Low-Sec (&gt;0.0)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {error && <div className="text-sm text-red-600">{error}</div>}

            <Button type="submit" disabled={isLoading || !selectedItem}>
              {isLoading ? "Berechne..." : "Routen berechnen"}
            </Button>
          </form>
        </CardContent>
      </Card>

      {routes.length > 0 && (
        <div className="space-y-4">
          <h2 className="text-2xl font-bold">Beste Verkaufsziele ({routes.length})</h2>
          {routes.map((route, index) => (
            <Card key={index}>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <span>{route.sell_station_name}</span>
                  <span className={getSecurityColor(route.min_route_security_status)}>
                    Sec {route.min_route_security_status.toFixed(1)}
                  </span>
                </CardTitle>
                <CardDescription>
                  {route.sell_system_name} • {route.route_jumps} {route.route_jumps === 1 ? "Jump" : "Jumps"}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <div className="text-muted-foreground">Buy Order</div>
                    <div className="text-lg font-bold">
                      {route.buy_order_price.toLocaleString("de-DE", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2,
                      })}{" "}
                      ISK
                    </div>
                  </div>

                  <div>
                    <div className="text-muted-foreground">Nach Steuern</div>
                    <div className="text-lg font-bold text-green-600">
                      {route.net_price_per_unit.toLocaleString("de-DE", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2,
                      })}{" "}
                      ISK
                    </div>
                  </div>

                  <div>
                    <div className="text-muted-foreground">Profit/Stück</div>
                    <div className="text-lg font-bold text-blue-600">
                      {route.profit_per_unit.toLocaleString("de-DE", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2,
                      })}{" "}
                      ISK
                    </div>
                  </div>

                  <div>
                    <div className="text-muted-foreground">Menge</div>
                    <div className="text-lg font-bold">{route.available_quantity.toLocaleString("de-DE")}</div>
                  </div>

                  <div className="col-span-2">
                    <div className="text-muted-foreground">Gesamtprofit</div>
                    <div className="text-2xl font-bold text-green-600">
                      {route.total_profit.toLocaleString("de-DE", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2,
                      })}{" "}
                      ISK
                    </div>
                  </div>

                  <div className="col-span-2">
                    <div className="text-muted-foreground mb-1">Steuersatz</div>
                    <div className="text-sm">
                      {(route.tax_rate * 100).toFixed(2)}% (Broker + Sales Tax)
                    </div>
                  </div>
                </div>

                <Button onClick={() => handleSetRoute(route)} className="mt-4 w-full">
                  Route setzen ({route.route_jumps} Jumps)
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {routes.length === 0 && !isLoading && selectedItem && (
        <Card>
          <CardContent className="p-6 text-center text-muted-foreground">
            Keine profitablen Routen gefunden. Versuche die Filter anzupassen.
          </CardContent>
        </Card>
      )}
    </div>
  );
}

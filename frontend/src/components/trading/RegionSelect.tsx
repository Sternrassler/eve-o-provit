"use client";

import { useEffect, useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { fetchRegions } from "@/lib/api-client";
import { Region } from "@/types/trading";
import { RegionStalenessIndicator } from "./RegionStalenessIndicator";
import { RegionRefreshButton } from "./RegionRefreshButton";

interface RegionSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  showStaleness?: boolean;
  showRefresh?: boolean;
  onRefreshComplete?: () => void;
  onRefreshStateChange?: (isRefreshing: boolean) => void;
}

export function RegionSelect({ 
  value, 
  onChange, 
  disabled,
  showStaleness = true,
  showRefresh = true,
  onRefreshComplete,
  onRefreshStateChange,
}: RegionSelectProps) {
  const [regions, setRegions] = useState<Region[]>([]);
  const [loading, setLoading] = useState(true);
  // True when the region list couldn't be loaded — shown loudly instead of
  // silently substituting a hardcoded mock list as if it were real.
  const [loadError, setLoadError] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    const loadRegions = async () => {
      setLoadError(false);
      try {
        const data = await fetchRegions();
        setRegions(data);
      } catch (error) {
        console.error("Failed to fetch regions:", error);
        setRegions([]);
        setLoadError(true);
      } finally {
        setLoading(false);
      }
    };

    loadRegions();
  }, []);

  const handleRefreshComplete = () => {
    setRefreshKey((prev) => prev + 1);
    onRefreshComplete?.();
  };

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Region</label>
      <div className="flex items-center gap-2">
        <Select value={value} onValueChange={onChange} disabled={disabled || loading || loadError}>
          <SelectTrigger className="w-full">
            <SelectValue
              placeholder={
                loading
                  ? "Lade Regionen..."
                  : loadError
                    ? "Regionen nicht verfügbar"
                    : "Region wählen..."
              }
            />
          </SelectTrigger>
          <SelectContent>
            {regions.map((region) => (
              <SelectItem key={region.id} value={region.id.toString()}>
                {region.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {showRefresh && (
          <RegionRefreshButton 
            regionId={value} 
            disabled={disabled || loading || !value}
            onRefreshComplete={handleRefreshComplete}
            onRefreshStateChange={onRefreshStateChange}
          />
        )}
      </div>
      {loadError && (
        <p className="text-xs text-destructive" role="alert">
          Regionen konnten nicht geladen werden. Bitte Seite neu laden.
        </p>
      )}
      {showStaleness && value && (
        <RegionStalenessIndicator
          key={refreshKey}
          regionId={value}
        />
      )}
    </div>
  );
}

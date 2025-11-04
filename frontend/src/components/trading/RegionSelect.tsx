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
import { regions as fallbackRegions } from "@/lib/mock-data/regions";
import { RegionStalenessIndicator } from "./RegionStalenessIndicator";
import { RegionRefreshButton } from "./RegionRefreshButton";

interface RegionSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  showStaleness?: boolean;
  showRefresh?: boolean;
  onRefreshComplete?: () => void;
}

export function RegionSelect({ 
  value, 
  onChange, 
  disabled,
  showStaleness = true,
  showRefresh = true,
  onRefreshComplete,
}: RegionSelectProps) {
  const [regions, setRegions] = useState<Region[]>(fallbackRegions);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    const loadRegions = async () => {
      try {
        const data = await fetchRegions();
        if (data && data.length > 0) {
          setRegions(data);
        }
      } catch (error) {
        console.error("Failed to fetch regions, using fallback:", error);
        // Keep fallback regions on error
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
        <Select value={value} onValueChange={onChange} disabled={disabled || loading}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder={loading ? "Lade Regionen..." : "Region wählen..."} />
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
          />
        )}
      </div>
      {showStaleness && value && (
        <RegionStalenessIndicator 
          key={refreshKey} 
          regionId={value} 
        />
      )}
    </div>
  );
}

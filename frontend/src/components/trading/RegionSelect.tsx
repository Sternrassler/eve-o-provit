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

interface RegionSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function RegionSelect({ value, onChange, disabled }: RegionSelectProps) {
  const [regions, setRegions] = useState<Region[]>(fallbackRegions);
  const [loading, setLoading] = useState(true);

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

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Region</label>
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
    </div>
  );
}

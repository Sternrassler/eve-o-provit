"use client";

import { useEffect, useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ships as fallbackShips } from "@/lib/mock-data/ships";
import { Ship } from "@/types/trading";
import { fetchCharacterShips } from "@/lib/api-client";

interface ShipSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  authenticated?: boolean;
  authHeader?: string | null;
}

export function ShipSelect({
  value,
  onChange,
  disabled,
  authenticated = false,
  authHeader = null,
}: ShipSelectProps) {
  const [ships, setShips] = useState<Ship[]>(fallbackShips);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const loadShips = async () => {
      if (!authenticated || !authHeader) {
        setShips(fallbackShips);
        return;
      }

      setLoading(true);
      try {
        const characterShips = await fetchCharacterShips(authHeader);
        if (characterShips && characterShips.length > 0) {
          setShips(characterShips);
        } else {
          // If user has no ships, use fallback
          setShips(fallbackShips);
        }
      } catch (error) {
        console.error("Failed to fetch character ships, using fallback:", error);
        setShips(fallbackShips);
      } finally {
        setLoading(false);
      }
    };

    loadShips();
  }, [authenticated, authHeader]);

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Schiff</label>
      <Select value={value} onValueChange={onChange} disabled={disabled || loading}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={loading ? "Lade Schiffe..." : "Schiff wählen..."} />
        </SelectTrigger>
        <SelectContent>
          {ships.map((ship) => (
            <SelectItem key={ship.type_id} value={ship.type_id.toString()}>
              {ship.name} ({(ship.cargo_capacity / 1000).toFixed(0)}k m³)
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

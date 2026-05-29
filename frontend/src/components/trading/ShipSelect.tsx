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
import { fetchCharacterShip, fetchCharacterShips } from "@/lib/api-client";

interface ShipSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  authenticated?: boolean;
}

export function ShipSelect({
  value,
  onChange,
  disabled,
  authenticated = false,
}: ShipSelectProps) {
  const [ships, setShips] = useState<Ship[]>(fallbackShips);
  // The active ship (the one being flown) — not necessarily in the hangar list,
  // so it's merged into the options separately to keep the current-ship default
  // selectable.
  const [activeShip, setActiveShip] = useState<Ship | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const loadShips = async () => {
      if (!authenticated) {
        setShips(fallbackShips);
        setActiveShip(null);
        return;
      }

      setLoading(true);
      try {
        const [characterShips, active] = await Promise.all([
          fetchCharacterShips().catch(() => null),
          fetchCharacterShip().catch(() => null),
        ]);
        if (characterShips && characterShips.length > 0) {
          setShips(characterShips);
        } else {
          setShips(fallbackShips);
        }
        if (active?.ship_type_id) {
          setActiveShip({
            type_id: active.ship_type_id,
            name: active.ship_type_name || active.ship_name,
            cargo_capacity: active.cargo_capacity,
          });
        }
      } finally {
        setLoading(false);
      }
    };

    loadShips();
  }, [authenticated]);

  // Active ship first (deduplicated), then the hangar/fallback ships.
  const options: Ship[] = [];
  const seen = new Set<number>();
  if (activeShip) {
    options.push(activeShip);
    seen.add(activeShip.type_id);
  }
  for (const s of ships) {
    if (!seen.has(s.type_id)) {
      options.push(s);
      seen.add(s.type_id);
    }
  }

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Schiff</label>
      <Select value={value} onValueChange={onChange} disabled={disabled || loading}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={loading ? "Lade Schiffe..." : "Schiff wählen..."} />
        </SelectTrigger>
        <SelectContent>
          {options.map((ship) => {
            // Format cargo capacity: show in m³ if < 1000, otherwise in k m³
            const cargoDisplay = ship.cargo_capacity >= 1000
              ? `${(ship.cargo_capacity / 1000).toFixed(1)}k m³`
              : `${Math.round(ship.cargo_capacity)} m³`;

            return (
              <SelectItem key={ship.type_id} value={ship.type_id.toString()}>
                {ship.name} ({cargoDisplay})
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
    </div>
  );
}

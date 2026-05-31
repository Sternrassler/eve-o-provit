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
  /** Called with the resolved Ship whose item_id matches `value` (or null when
   *  no option matches). Lets the parent read the selected instance's effective
   *  cargo for the optimizer override. */
  onSelect?: (ship: Ship | null) => void;
}

export function ShipSelect({
  value,
  onChange,
  disabled,
  authenticated = false,
  onSelect,
}: ShipSelectProps) {
  // Unauthenticated users get a generic example list (a legitimate default —
  // there is no character to load). Authenticated users get their real hangar;
  // on a fetch failure we show NO ships and a loud error instead of passing off
  // the generic list as their fleet.
  const [ships, setShips] = useState<Ship[]>(authenticated ? [] : fallbackShips);
  // The active ship (the one being flown) — not necessarily in the hangar list,
  // so it's merged into the options separately to keep the current-ship default
  // selectable.
  const [activeShip, setActiveShip] = useState<Ship | null>(null);
  const [loading, setLoading] = useState(false);
  // True when loading the authenticated character's ships failed — surfaced as a
  // prominent error so wrong/empty data is never read as the real fleet.
  const [loadError, setLoadError] = useState(false);

  useEffect(() => {
    const loadShips = async () => {
      if (!authenticated) {
        setShips(fallbackShips);
        setActiveShip(null);
        setLoadError(false);
        return;
      }

      setLoading(true);
      let failed = false;
      try {
        const [characterShips, active] = await Promise.all([
          fetchCharacterShips().catch(() => {
            failed = true;
            return null;
          }),
          fetchCharacterShip().catch(() => {
            failed = true;
            return null;
          }),
        ]);
        // On failure show NO ships (never the generic mock list as if it were
        // the character's). An empty-but-successful hangar is also just empty.
        setShips(characterShips ?? []);
        if (active?.ship_type_id) {
          setActiveShip({
            item_id: active.ship_item_id,
            type_id: active.ship_type_id,
            name: active.ship_type_name || active.ship_name,
            cargo_capacity: active.cargo_capacity,
            effective_cargo_capacity: active.effective_cargo_capacity,
            effective_cargo_unavailable: active.effective_cargo_unavailable,
          });
        } else {
          setActiveShip(null);
        }
      } finally {
        setLoadError(failed);
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
    seen.add(activeShip.item_id);
  }
  for (const s of ships) {
    if (!seen.has(s.item_id)) {
      options.push(s);
      seen.add(s.item_id);
    }
  }

  // Surface the currently selected instance to the parent. Keyed on `value` and
  // the inputs that determine the option list (`ships` + `activeShip`), so the
  // parent always has the up-to-date selection even as the hangar loads.
  useEffect(() => {
    if (!onSelect) return;
    const selected =
      options.find((s) => s.item_id.toString() === value) ?? null;
    onSelect(selected);
    // `options` is rebuilt every render; depend on its inputs instead.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, ships, activeShip, onSelect]);

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Schiff</label>
      <Select value={value} onValueChange={onChange} disabled={disabled || loading}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={loading ? "Lade Schiffe..." : "Schiff wählen..."} />
        </SelectTrigger>
        <SelectContent>
          {options.map((ship) => {
            // Prefer the effective cargo (matches what the optimizer uses and
            // what EVE shows in-game). When the backend's fitting enrichment
            // ERRORED, the effective volume is genuinely unknown — say so
            // instead of passing off the base hull as the fitted value. Only a
            // ship that truly has no cargo-expander shows the bare "Basis" hull.
            const effective = ship.effective_cargo_capacity;
            const useEffective = effective != null && effective > 0;
            const value = useEffective ? effective : ship.cargo_capacity;
            const cargoFmt =
              value >= 1000
                ? `${(value / 1000).toFixed(1)}k m³`
                : `${Math.round(value)} m³`;
            const cargoDisplay = ship.effective_cargo_unavailable
              ? `Basis ${cargoFmt} — fitted unbekannt`
              : useEffective
                ? cargoFmt
                : `Basis ${cargoFmt}`;

            return (
              <SelectItem key={ship.item_id.toString()} value={ship.item_id.toString()}>
                {ship.name} ({cargoDisplay})
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
      {loadError && (
        <p className="text-xs text-destructive" role="alert">
          Deine Schiffe konnten nicht geladen werden. Bitte neu laden — es wird
          keine Ersatzliste angezeigt, um falsche Schiffsdaten zu vermeiden.
        </p>
      )}
    </div>
  );
}

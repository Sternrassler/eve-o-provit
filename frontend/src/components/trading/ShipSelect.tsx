"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ships, authenticatedShips } from "@/lib/mock-data/ships";

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
  const shipList = authenticated ? authenticatedShips : ships;

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Schiff</label>
      <Select value={value} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Schiff wählen..." />
        </SelectTrigger>
        <SelectContent>
          {shipList.map((ship) => (
            <SelectItem key={ship.type_id} value={ship.type_id.toString()}>
              {ship.name} ({(ship.cargo_capacity / 1000).toFixed(0)}k m³)
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

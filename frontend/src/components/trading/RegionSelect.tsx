"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { regions } from "@/lib/mock-data/regions";

interface RegionSelectProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function RegionSelect({ value, onChange, disabled }: RegionSelectProps) {
  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">Region</label>
      <Select value={value} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Region wählen..." />
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

"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchCharacterFitting } from "@/lib/api-client";
import { CharacterFittingResponse, FittedModule } from "@/types/character";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { InfoIcon } from "lucide-react";

interface ShipFittingCardProps {
  characterId: number;
  shipTypeId: number;
  authHeader: string | null;
  className?: string;
}

/**
 * ShipFittingCard displays character's current ship fitting with bonus calculations
 * Shows:
 * - Fitted modules grouped by slot (High/Med/Low/Rig)
 * - Aggregated bonuses (cargo, warp speed, inertia)
 * - Cache status
 */
export function ShipFittingCard({
  characterId,
  shipTypeId,
  authHeader,
  className,
}: ShipFittingCardProps) {
  const [fitting, setFitting] = useState<CharacterFittingResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadFitting = async () => {
      if (!authHeader || !characterId || !shipTypeId) {
        setFitting(null);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const data = await fetchCharacterFitting(authHeader, characterId, shipTypeId);
        setFitting(data);
      } catch (err) {
        console.error("Failed to fetch fitting:", err);
        setError(err instanceof Error ? err.message : "Unknown error");
        setFitting(null);
      } finally {
        setLoading(false);
      }
    };

    loadFitting();
  }, [characterId, shipTypeId, authHeader]);

  if (loading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>Schiff-Fitting</CardTitle>
          <CardDescription>Lade Fitting-Daten...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>Schiff-Fitting</CardTitle>
          <CardDescription>Fehler beim Laden</CardDescription>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <InfoIcon className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  if (!fitting || fitting.fitted_modules.length === 0) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>Schiff-Fitting</CardTitle>
          <CardDescription>Keine Module gefittet</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Dieses Schiff hat keine Module in den relevanten Slots (Cargo, Warp, Inertia).
          </p>
        </CardContent>
      </Card>
    );
  }

  // Group modules by slot type
  const slotGroups = groupModulesBySlot(fitting.fitted_modules);

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Schiff-Fitting</CardTitle>
            <CardDescription>
              {fitting.fitted_modules.length} Module gefittet
              {fitting.cached && <span className="ml-2 text-xs">(Cache)</span>}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Bonuses Summary */}
        <div className="grid grid-cols-1 gap-3">
          <BonusCard
            label="Cargo Bonus"
            value={fitting.bonuses.cargo_bonus_m3}
            effectiveValue={fitting.effective_cargo_m3}
            unit="m³"
            positive={fitting.bonuses.cargo_bonus_m3 > 0}
          />
          <BonusCard
            label="Warp Speed"
            value={(fitting.bonuses.warp_speed_multiplier - 1) * 100}
            effectiveValue={fitting.warp_speed_au_s}
            unit="AU/s"
            positive={fitting.bonuses.warp_speed_multiplier > 1}
          />
          <BonusCard
            label="Agility"
            value={(1 - fitting.bonuses.inertia_modifier) * 100}
            effectiveValue={fitting.align_time_seconds}
            unit="s"
            positive={fitting.bonuses.inertia_modifier < 1}
          />
        </div>

        {/* Fitted Modules by Slot */}
        <div className="space-y-3">
          <h3 className="text-sm font-medium">Gefittete Module</h3>
          {Object.entries(slotGroups).map(([slotType, modules]) => (
            <div key={slotType} className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase">
                {slotType}
              </h4>
              <div className="space-y-1">
                {modules.map((module, idx) => (
                  <div
                    key={`${module.slot}-${idx}`}
                    className="flex items-center justify-between text-sm"
                  >
                    <span className="text-muted-foreground">{module.type_name}</span>
                    <Badge variant="outline" className="text-xs">
                      {module.slot}
                    </Badge>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * Group modules by slot type (High/Med/Low/Rig)
 */
function groupModulesBySlot(modules: FittedModule[]): Record<string, FittedModule[]> {
  const groups: Record<string, FittedModule[]> = {
    "High Slots": [],
    "Med Slots": [],
    "Low Slots": [],
    "Rig Slots": [],
  };

  modules.forEach((module) => {
    if (module.slot.startsWith("HiSlot")) {
      groups["High Slots"].push(module);
    } else if (module.slot.startsWith("MedSlot")) {
      groups["Med Slots"].push(module);
    } else if (module.slot.startsWith("LoSlot")) {
      groups["Low Slots"].push(module);
    } else if (module.slot.startsWith("RigSlot")) {
      groups["Rig Slots"].push(module);
    }
  });

  // Remove empty groups
  return Object.fromEntries(
    Object.entries(groups).filter(([_slot, mods]) => mods.length > 0)
  );
}

/**
 * BonusCard displays a single bonus metric with effective value
 */
function BonusCard({
  label,
  value,
  effectiveValue,
  unit,
  positive,
}: {
  label: string;
  value: number;
  effectiveValue: number;
  unit: string;
  positive: boolean;
}) {
  // Display bonus as percentage for cargo/warp/agility
  const displayBonus = Math.abs(value).toLocaleString("de-DE", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 1,
  });

  // Display effective value with appropriate precision based on unit
  const decimals = unit === "s" ? 2 : unit === "AU/s" ? 2 : 1;
  const displayEffective = effectiveValue.toLocaleString("de-DE", {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });

  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  const colorClass = positive
    ? "text-green-600 dark:text-green-400"
    : value < 0
    ? "text-red-600 dark:text-red-400"
    : "text-muted-foreground";

  return (
    <div className="rounded-lg border p-3">
      <div className="text-xs text-muted-foreground mb-1">{label}</div>
      <div className={`text-2xl font-semibold ${colorClass}`}>
        {displayEffective}
        <span className="text-base ml-1">{unit}</span>
      </div>
      <div className="text-xs text-muted-foreground mt-1">
        {sign}{displayBonus}%
      </div>
    </div>
  );
}

"use client";

import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useCurrentShip } from "@/lib/use-current-ship";

interface CurrentShipCardProps {
  className?: string;
}

export function CurrentShipCard({ className }: CurrentShipCardProps) {
  const { ship, isLoading, error, refresh } = useCurrentShip();

  const cargoDisplay = (() => {
    if (!ship) return null;
    const eff = ship.effective_cargo_capacity;
    const useEffective = eff != null && eff > 0;
    const value = useEffective ? eff : ship.cargo_capacity;
    const cargoFmt =
      value >= 1000
        ? `${(value / 1000).toFixed(1)}k m³`
        : `${Math.round(value)} m³`;
    return ship.effective_cargo_unavailable
      ? `Basis ${cargoFmt} — fitted unbekannt`
      : useEffective
        ? cargoFmt
        : `Basis ${cargoFmt}`;
  })();

  return (
    <div className={`space-y-1 ${className ?? ""}`.trim()}>
      <div className="flex items-center gap-1">
        <label className="text-sm font-medium">Schiff</label>
        <Button
          variant="ghost"
          size="icon"
          className="h-5 w-5"
          onClick={refresh}
          disabled={isLoading}
          title="Aktuelles Schiff neu laden"
        >
          <RefreshCw className={`h-3 w-3 ${isLoading ? "animate-spin" : ""}`} />
        </Button>
      </div>
      {error ? (
        <p role="alert" className="text-sm text-destructive">
          Aktuelles Schiff konnte nicht geladen werden
        </p>
      ) : (
        <p className="text-sm text-muted-foreground">
          {!ship
            ? isLoading
              ? "Lade aktuelles Schiff…"
              : "Kein aktives Schiff"
            : `${ship.ship_name} (${cargoDisplay})`}
        </p>
      )}
    </div>
  );
}

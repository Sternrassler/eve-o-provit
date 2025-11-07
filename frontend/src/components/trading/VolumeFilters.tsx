"use client";

import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { TrendingUp, Clock, BarChart3 } from "lucide-react";

interface VolumeFiltersProps {
  includeVolumeMetrics: boolean;
  onIncludeVolumeMetricsChange: (value: boolean) => void;
  minDailyVolume: number;
  onMinDailyVolumeChange: (value: number) => void;
  maxLiquidationDays: number;
  onMaxLiquidationDaysChange: (value: number) => void;
}

export function VolumeFilters({
  includeVolumeMetrics,
  onIncludeVolumeMetricsChange,
  minDailyVolume,
  onMinDailyVolumeChange,
  maxLiquidationDays,
  onMaxLiquidationDaysChange,
}: VolumeFiltersProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BarChart3 className="size-5" />
          Volume & Liquidität
        </CardTitle>
        <CardDescription>
          Filtere nach Handelsvolumen und Liquidität für bessere Kapitalumschlagrate
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Enable Volume Metrics */}
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label className="text-base">Volume-Metriken aktivieren</Label>
            <p className="text-sm text-muted-foreground">
              Zeigt Handelsvolumen, Liquidation und Daily Profit an
            </p>
          </div>
          <Switch
            checked={includeVolumeMetrics}
            onCheckedChange={onIncludeVolumeMetricsChange}
          />
        </div>

        {/* Filters - only shown when volume metrics are enabled */}
        {includeVolumeMetrics && (
          <>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label className="flex items-center gap-2">
                  <TrendingUp className="size-4" />
                  Min. Tägliches Volume
                </Label>
                <span className="text-sm font-medium">
                  {minDailyVolume === 0 ? "Kein Minimum" : `${minDailyVolume} Items/Tag`}
                </span>
              </div>
              <Slider
                value={[minDailyVolume]}
                onValueChange={(values) => onMinDailyVolumeChange(values[0])}
                min={0}
                max={1000}
                step={50}
                className="w-full"
              />
              <p className="text-xs text-muted-foreground">
                Filtert Items mit zu geringem täglichen Handelsvolumen aus
              </p>
            </div>

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label className="flex items-center gap-2">
                  <Clock className="size-4" />
                  Max. Liquidationszeit
                </Label>
                <span className="text-sm font-medium">
                  {maxLiquidationDays === 30 ? "Kein Limit" : `${maxLiquidationDays} Tage`}
                </span>
              </div>
              <Slider
                value={[maxLiquidationDays]}
                onValueChange={(values) => onMaxLiquidationDaysChange(values[0])}
                min={1}
                max={30}
                step={1}
                className="w-full"
              />
              <p className="text-xs text-muted-foreground">
                Zeit bis komplette Menge verkauft werden kann (bei 10% Marktanteil)
              </p>
            </div>

            <div className="rounded-lg bg-muted/50 p-3 space-y-1">
              <p className="text-xs font-medium">💡 Empfehlung:</p>
              <p className="text-xs text-muted-foreground">
                Min. Volume: 100 Items/Tag, Max. Liquidation: 7 Tage für optimalen Kapitalumschlag
              </p>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

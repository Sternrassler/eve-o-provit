"use client";

import { useState } from "react";
import { TradingTabs } from "@/components/trading/TradingTabs";
import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { CurrentShipCard } from "@/components/trading/CurrentShipCard";
import { HaulingRouteList } from "@/components/trading/HaulingRouteList";
import { SkillsAppliedPanel } from "@/components/trading/SkillsAppliedPanel";
import { findHaulingRoutes } from "@/lib/api-client";
import { HaulingRequest } from "@/types/trading";
import { useCurrentShip } from "@/lib/use-current-ship";
import { CharacterShip } from "@/types/character";
import { Loader2 } from "lucide-react";

const DEFAULT_CAPITAL = 500_000_000;
const DEFAULT_MAX_ROUTES = 15;

interface HaulingFormState {
  capital: number;
  avoidLowSec: boolean;
}

const defaultForm: HaulingFormState = {
  capital: DEFAULT_CAPITAL,
  avoidLowSec: true,
};

function buildRequest(
  form: HaulingFormState,
  ship: CharacterShip,
): HaulingRequest {
  // Pass the instance's exact effective cargo only when known, positive and not
  // flagged unavailable — otherwise omit it and let the backend recompute.
  const cargoCapacity =
    !ship.effective_cargo_unavailable &&
    ship.effective_cargo_capacity != null &&
    ship.effective_cargo_capacity > 0
      ? ship.effective_cargo_capacity
      : undefined;

  return {
    origin_region_id: 0, // backend uses the character's current region
    ship_type_id: ship.ship_type_id,
    capital: form.capital,
    avoid_low_sec: form.avoidLowSec,
    max_routes: DEFAULT_MAX_ROUTES,
    cargo_capacity: cargoCapacity,
  };
}

function HaulingPageContent() {
  const { isAuthenticated } = useAuth();
  const [form, setForm] = useState<HaulingFormState>(defaultForm);

  // The ship is always the character's current ship — no longer user-selectable.
  const { ship: currentShip } = useCurrentShip();

  const haulingMutation = useMutation({
    mutationFn: (req: HaulingRequest) => findHaulingRoutes(req),
  });

  const handleSubmit = () => {
    if (!currentShip) return; // gated by the submit button; guard for safety
    haulingMutation.mutate(buildRequest(form, currentShip));
  };

  const apiError = haulingMutation.isError
    ? haulingMutation.error instanceof Error
      ? haulingMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  const disabled = !isAuthenticated || haulingMutation.isPending;

  return (
    <div className="container mx-auto px-4 py-8">
      <TradingTabs />
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Neighborhood Hauling Routes</h1>
        <p className="text-muted-foreground">
          Profitable Station-zu-Station Hauling-Routen in deiner aktuellen
          Region und den angrenzenden Regionen
        </p>
      </div>

      {!isAuthenticated && (
        <Alert className="mb-6">
          <AlertTitle>Login erforderlich</AlertTitle>
          <AlertDescription>
            Melde dich per EVE SSO an, um Hauling-Routen rund um deine aktuelle
            Position zu finden.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-[340px_1fr]">
        {/* Input form */}
        <form
          className="space-y-4 rounded-lg border p-4"
          onSubmit={(e) => {
            e.preventDefault();
            handleSubmit();
          }}
        >
          <CurrentShipCard />

          <div className="space-y-2">
            <Label htmlFor="capital">Kapital (ISK)</Label>
            <Input
              id="capital"
              type="number"
              min={0}
              value={form.capital}
              disabled={disabled}
              onChange={(e) =>
                setForm((s) => ({ ...s, capital: Number(e.target.value) }))
              }
            />
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="avoid-low-sec"
              checked={form.avoidLowSec}
              disabled={disabled}
              onCheckedChange={(c) =>
                setForm((s) => ({ ...s, avoidLowSec: c === true }))
              }
            />
            <Label htmlFor="avoid-low-sec" className="font-normal">
              Low-Sec meiden
            </Label>
          </div>

          <Button
            type="submit"
            className="w-full"
            // Block submission until the current ship is loaded — otherwise the
            // request would carry ship_type_id: 0.
            disabled={disabled || !currentShip}
          >
            {haulingMutation.isPending && (
              <Loader2 className="mr-2 size-4 animate-spin" />
            )}
            {haulingMutation.isPending ? "Suche Routen..." : "Routen finden"}
          </Button>
        </form>

        {/* Results */}
        <div className="space-y-6">
          {haulingMutation.isPending && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Suche profitable Hauling-Routen im Umkreis...
            </div>
          )}

          {apiError && (
            <Alert variant="destructive">
              <AlertTitle>Fehler</AlertTitle>
              <AlertDescription className="flex items-center justify-between gap-4">
                <span>{apiError}</span>
                <Button variant="outline" size="sm" onClick={handleSubmit}>
                  Erneut versuchen
                </Button>
              </AlertDescription>
            </Alert>
          )}

          {haulingMutation.isSuccess && haulingMutation.data && (
            <div className="space-y-6">
              <HaulingRouteList routes={haulingMutation.data.routes} />
              <SkillsAppliedPanel
                skills={haulingMutation.data.skills_applied}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function HaulingPage() {
  return (
    <ErrorBoundary>
      <HaulingPageContent />
    </ErrorBoundary>
  );
}

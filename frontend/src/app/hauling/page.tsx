"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { ShipSelect } from "@/components/trading/ShipSelect";
import { HaulingRouteList } from "@/components/trading/HaulingRouteList";
import { SkillsAppliedPanel } from "@/components/trading/SkillsAppliedPanel";
import { findHaulingRoutes } from "@/lib/api-client";
import { HaulingRequest } from "@/types/trading";
import { Loader2 } from "lucide-react";

const DEFAULT_SHIP = "649";
const DEFAULT_CAPITAL = 500_000_000;
const DEFAULT_MAX_ROUTES = 15;

interface HaulingFormState {
  ship: string;
  capital: number;
  avoidLowSec: boolean;
}

const defaultForm: HaulingFormState = {
  ship: DEFAULT_SHIP,
  capital: DEFAULT_CAPITAL,
  avoidLowSec: true,
};

function buildRequest(form: HaulingFormState): HaulingRequest {
  return {
    origin_region_id: 0, // backend uses the character's current region
    ship_type_id: parseInt(form.ship, 10),
    capital: form.capital,
    avoid_low_sec: form.avoidLowSec,
    max_routes: DEFAULT_MAX_ROUTES,
  };
}

function HaulingPageContent() {
  const { isAuthenticated } = useAuth();
  const [form, setForm] = useState<HaulingFormState>(defaultForm);

  const haulingMutation = useMutation({
    mutationFn: (req: HaulingRequest) => findHaulingRoutes(req),
  });

  const handleSubmit = () => {
    haulingMutation.mutate(buildRequest(form));
  };

  const apiError = haulingMutation.isError
    ? haulingMutation.error instanceof Error
      ? haulingMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  const disabled = !isAuthenticated || haulingMutation.isPending;

  return (
    <div className="container mx-auto px-4 py-8">
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
          <ShipSelect
            value={form.ship}
            onChange={(v) => setForm((s) => ({ ...s, ship: v }))}
            disabled={disabled}
            authenticated={isAuthenticated}
          />

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
            disabled={disabled || !form.ship}
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

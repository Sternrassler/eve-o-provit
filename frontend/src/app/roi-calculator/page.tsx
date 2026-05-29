"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  PortfolioInputForm,
  PortfolioFormState,
} from "@/components/trading/PortfolioInputForm";
import { PortfolioResultTable } from "@/components/trading/PortfolioResultTable";
import { DiversificationScore } from "@/components/trading/DiversificationScore";
import { SkillsAppliedPanel } from "@/components/trading/SkillsAppliedPanel";
import {
  fetchCharacterLocation,
  fetchCharacterShip,
  optimizePortfolio,
} from "@/lib/api-client";
import { PortfolioRequest } from "@/types/trading";
import { Loader2 } from "lucide-react";

const DEFAULT_REGION = "10000002"; // The Forge
const DEFAULT_SHIP = "649";

const defaultForm: PortfolioFormState = {
  region: DEFAULT_REGION,
  ship: DEFAULT_SHIP,
  capital: 500_000_000,
  timeBudgetMin: 120,
  liquidityCapPct: 10,
  maxItemPct: 30,
  allowHighSec: true,
  allowLowSec: false,
  allowNullSec: false,
};

function buildRequest(form: PortfolioFormState): PortfolioRequest {
  const secZones: string[] = [];
  if (form.allowHighSec) secZones.push("high");
  if (form.allowLowSec) secZones.push("low");
  if (form.allowNullSec) secZones.push("null");

  return {
    region_id: parseInt(form.region, 10),
    ship_type_id: parseInt(form.ship, 10),
    capital: form.capital,
    time_budget_min: form.timeBudgetMin,
    liquidity_cap_pct: form.liquidityCapPct,
    max_item_pct: form.maxItemPct,
    sec_zones: secZones,
  };
}

function ROICalculatorContent() {
  const { isAuthenticated } = useAuth();
  const [form, setForm] = useState<PortfolioFormState>(defaultForm);
  // null = no manual choice yet; effective region/ship are derived below.
  const [regionOverride, setRegionOverride] = useState<string | null>(null);
  const [shipOverride, setShipOverride] = useState<string | null>(null);

  // Load the character's current location + ship to pre-fill region/ship.
  const { data: characterData } = useQuery({
    queryKey: ["characterData", isAuthenticated],
    queryFn: async () => {
      const [location, ship] = await Promise.all([
        fetchCharacterLocation(),
        fetchCharacterShip(),
      ]);
      return { location, ship };
    },
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });

  // Effective values: manual override > current character data > default.
  // Purely derived (no effect), so the user's choice stays sticky.
  const effectiveForm: PortfolioFormState = {
    ...form,
    region:
      regionOverride ??
      characterData?.location.region_id?.toString() ??
      DEFAULT_REGION,
    ship:
      shipOverride ??
      characterData?.ship.ship_type_id?.toString() ??
      DEFAULT_SHIP,
  };

  const handleFormChange = (next: PortfolioFormState) => {
    if (next.region !== effectiveForm.region) setRegionOverride(next.region);
    if (next.ship !== effectiveForm.ship) setShipOverride(next.ship);
    setForm(next);
  };

  const optimizeMutation = useMutation({
    mutationFn: (req: PortfolioRequest) => optimizePortfolio(req),
  });

  const handleSubmit = () => {
    optimizeMutation.mutate(buildRequest(effectiveForm));
  };

  const apiError = optimizeMutation.isError
    ? optimizeMutation.error instanceof Error
      ? optimizeMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">
          ROI Calculator & Capital Optimizer
        </h1>
        <p className="text-muted-foreground">
          Verteile dein Kapital optimal über mehrere Items für maximalen
          Tagesgewinn
        </p>
      </div>

      {!isAuthenticated && (
        <Alert className="mb-6">
          <AlertTitle>Login erforderlich</AlertTitle>
          <AlertDescription>
            Melde dich per EVE SSO an, um eine skill-bereinigte
            Kapital-Allokation zu berechnen.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-[340px_1fr]">
        {/* Input form */}
        <PortfolioInputForm
          state={effectiveForm}
          onChange={handleFormChange}
          onSubmit={handleSubmit}
          disabled={!isAuthenticated || optimizeMutation.isPending}
          loading={optimizeMutation.isPending}
          authenticated={isAuthenticated}
        />

        {/* Results */}
        <div className="space-y-6">
          {optimizeMutation.isPending && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Berechne optimale Kapital-Allokation...
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

          {optimizeMutation.isSuccess && optimizeMutation.data && (
            <div className="space-y-6">
              <PortfolioResultTable result={optimizeMutation.data} />
              <div className="grid gap-6 md:grid-cols-2">
                <DiversificationScore
                  score={optimizeMutation.data.diversification_score}
                />
                <SkillsAppliedPanel
                  skills={optimizeMutation.data.skills_applied}
                />
              </div>
              <p className="text-sm text-muted-foreground">
                Genutzte Zeit: {optimizeMutation.data.time_used_min} Min/Tag
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function ROICalculatorPage() {
  return (
    <ErrorBoundary>
      <ROICalculatorContent />
    </ErrorBoundary>
  );
}

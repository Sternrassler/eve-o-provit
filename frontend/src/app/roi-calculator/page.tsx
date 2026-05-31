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
  fetchCharacterWallet,
  optimizePortfolio,
} from "@/lib/api-client";
import { PortfolioRequest, Ship } from "@/types/trading";
import { buildPortfolioRequest } from "@/lib/portfolio-request";
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

function ROICalculatorContent() {
  const { isAuthenticated } = useAuth();
  const [form, setForm] = useState<PortfolioFormState>(defaultForm);
  // null = no manual choice yet; effective region/ship/capital are derived below.
  const [regionOverride, setRegionOverride] = useState<string | null>(null);
  const [shipOverride, setShipOverride] = useState<string | null>(null);
  const [capitalOverride, setCapitalOverride] = useState<number | null>(null);
  // The ship instance currently picked in the dropdown (reported by ShipSelect).
  // Its effective cargo is sent as the optimizer override.
  const [selectedShip, setSelectedShip] = useState<Ship | null>(null);

  // Load the character's current location + ship + wallet to pre-fill the form.
  const { data: characterData } = useQuery({
    queryKey: ["characterData", isAuthenticated],
    queryFn: async () => {
      const [location, ship, wallet] = await Promise.all([
        fetchCharacterLocation(),
        fetchCharacterShip(),
        // Wallet needs the (newer) wallet scope; tolerate its absence so a
        // character authorized before the scope was added still works — but
        // capture the failure so the UI can say the capital is a placeholder
        // rather than silently presenting the default as the real balance.
        fetchCharacterWallet().then(
          (value) => ({ value }),
          (err): { error: string } => ({
            error:
              err instanceof Error
                ? err.message
                : "Wallet konnte nicht geladen werden",
          }),
        ),
      ]);
      return { location, ship, wallet };
    },
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });

  const wallet = characterData?.wallet;
  const walletValue = wallet && "value" in wallet ? wallet.value : null;
  const walletError = wallet && "error" in wallet ? wallet.error : null;

  // Effective values: manual override > current character data > default.
  // Purely derived (no effect), so the user's choice stays sticky.
  const effectiveForm: PortfolioFormState = {
    ...form,
    region:
      regionOverride ??
      characterData?.location.region_id?.toString() ??
      DEFAULT_REGION,
    // The dropdown value is a ship instance's item_id, so the prefill must be
    // the active ship's item_id (not its type_id) to select the right option.
    ship:
      shipOverride ??
      characterData?.ship.ship_item_id?.toString() ??
      DEFAULT_SHIP,
    capital:
      capitalOverride ??
      (walletValue != null ? Math.floor(walletValue) : form.capital),
  };

  // The capital is the default placeholder ONLY because the wallet fetch failed
  // and the user hasn't entered a value — surface that so it's not mistaken for
  // the real balance.
  const showWalletWarning = walletError != null && capitalOverride == null;

  const handleFormChange = (next: PortfolioFormState) => {
    if (next.region !== effectiveForm.region) setRegionOverride(next.region);
    if (next.ship !== effectiveForm.ship) setShipOverride(next.ship);
    if (next.capital !== effectiveForm.capital)
      setCapitalOverride(next.capital);
    setForm(next);
  };

  const optimizeMutation = useMutation({
    mutationFn: (req: PortfolioRequest) => optimizePortfolio(req),
  });

  const handleSubmit = () => {
    optimizeMutation.mutate(buildPortfolioRequest(effectiveForm, selectedShip));
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

      {showWalletWarning && (
        <Alert variant="destructive" className="mb-6">
          <AlertTitle>Wallet-Guthaben nicht geladen</AlertTitle>
          <AlertDescription>
            {walletError} — es wird das Standardkapital verwendet. Bitte das
            Kapital manuell setzen; der Wert ist <strong>nicht</strong> dein
            echtes Guthaben.
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
          onShipSelect={setSelectedShip}
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

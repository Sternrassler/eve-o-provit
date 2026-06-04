"use client";

import { useState } from "react";
import { TradingTabs } from "@/components/trading/TradingTabs";
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
  fetchCharacterWallet,
  optimizePortfolio,
} from "@/lib/api-client";
import { PortfolioRequest, Ship } from "@/types/trading";
import { buildPortfolioRequest } from "@/lib/portfolio-request";
import { useCurrentShip } from "@/lib/use-current-ship";
import { Loader2 } from "lucide-react";

const DEFAULT_REGION = "10000002"; // The Forge

const defaultForm: PortfolioFormState = {
  region: DEFAULT_REGION,
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
  // null = no manual choice yet; effective region/capital are derived below.
  const [regionOverride, setRegionOverride] = useState<string | null>(null);
  const [capitalOverride, setCapitalOverride] = useState<number | null>(null);

  // The ship is always the character's current ship — no longer user-selectable.
  const { ship: currentShip } = useCurrentShip();
  const shipForRequest: Ship | null = currentShip
    ? {
        item_id: currentShip.ship_item_id,
        type_id: currentShip.ship_type_id,
        name: currentShip.ship_name,
        cargo_capacity: currentShip.cargo_capacity,
        effective_cargo_capacity: currentShip.effective_cargo_capacity,
        effective_cargo_unavailable: currentShip.effective_cargo_unavailable,
      }
    : null;

  // Load the character's current location + wallet to pre-fill the form.
  const { data: characterData } = useQuery({
    queryKey: ["characterData", isAuthenticated],
    queryFn: async () => {
      const [location, wallet] = await Promise.all([
        fetchCharacterLocation(),
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
      return { location, wallet };
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
    if (next.capital !== effectiveForm.capital)
      setCapitalOverride(next.capital);
    setForm(next);
  };

  const optimizeMutation = useMutation({
    mutationFn: (req: PortfolioRequest) => optimizePortfolio(req),
  });

  const handleSubmit = () => {
    optimizeMutation.mutate(
      buildPortfolioRequest(effectiveForm, shipForRequest),
    );
  };

  const apiError = optimizeMutation.isError
    ? optimizeMutation.error instanceof Error
      ? optimizeMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  return (
    <div className="container mx-auto px-4 py-8">
      <TradingTabs />
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
          // Block submission until the current ship is loaded — otherwise the
          // request would carry ship_type_id: 0.
          submitDisabled={!shipForRequest}
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

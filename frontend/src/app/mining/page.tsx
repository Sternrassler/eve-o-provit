"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CurrentShipCard } from "@/components/trading/CurrentShipCard";
import { OreRankingTable } from "@/components/trading/OreRankingTable";
import { fetchOreRanking } from "@/lib/api-client";
import { OreRankingRequest } from "@/types/trading";
import { Loader2 } from "lucide-react";

type SecBand = "high" | "low" | "null";

const SEC_BAND_LABELS: Record<SecBand, string> = {
  high: "High-Sec",
  low: "Low-Sec",
  null: "Null-Sec",
};

function MiningPageContent() {
  const { isAuthenticated } = useAuth();
  const [secBand, setSecBand] = useState<SecBand>("high");

  const miningMutation = useMutation({
    mutationFn: (req: OreRankingRequest) => fetchOreRanking(req),
  });

  const handleSubmit = () => {
    miningMutation.mutate({ region_id: 0, sec_band: secBand });
  };

  const apiError = miningMutation.isError
    ? miningMutation.error instanceof Error
      ? miningMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  const disabled = !isAuthenticated || miningMutation.isPending;

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Mining</h1>
        <p className="text-muted-foreground">
          Erze nach ISK/Stunde ranken — Roh-Verkauf vs. Reprozessieren
        </p>
      </div>

      {!isAuthenticated && (
        <Alert className="mb-6">
          <AlertTitle>Login erforderlich</AlertTitle>
          <AlertDescription>
            Melde dich per EVE SSO an, um Erze in deiner aktuellen Region zu
            ranken.
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
            <Label htmlFor="sec-band">Sicherheitsklasse</Label>
            <Select
              value={secBand}
              onValueChange={(v) => setSecBand(v as SecBand)}
              disabled={disabled}
            >
              <SelectTrigger id="sec-band">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="high">{SEC_BAND_LABELS.high}</SelectItem>
                <SelectItem value="low">{SEC_BAND_LABELS.low}</SelectItem>
                <SelectItem value="null">{SEC_BAND_LABELS.null}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Button type="submit" className="w-full" disabled={disabled}>
            {miningMutation.isPending && (
              <Loader2 className="mr-2 size-4 animate-spin" />
            )}
            {miningMutation.isPending ? "Berechne..." : "Erze berechnen"}
          </Button>
        </form>

        {/* Results */}
        <div className="space-y-6">
          {miningMutation.isPending && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Berechne Erz-Ranking...
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

          {miningMutation.isSuccess && miningMutation.data && (
            <div className="space-y-4">
              {miningMutation.data.no_mining_setup && (
                <Alert>
                  <AlertTitle>Kein Mining-Setup</AlertTitle>
                  <AlertDescription>
                    Dein aktuelles Schiff hat kein Mining-Setup — ISK/h kann
                    nicht berechnet werden
                  </AlertDescription>
                </Alert>
              )}
              <OreRankingTable rows={miningMutation.data.rows} />
              <p className="text-xs text-muted-foreground">
                Hinweis: ISK/h berücksichtigt Schiffs-/Skill-Boni und (best-case) Erz-Crystals.
                Zeilen mit ≈ sind Schätzwerte (Schiffs-Bonus oder Crystal nicht auflösbar). Die
                Roh-vs-Refine-Entscheidung ist davon unberührt.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function MiningPage() {
  return (
    <ErrorBoundary>
      <MiningPageContent />
    </ErrorBoundary>
  );
}

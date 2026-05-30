"use client";

import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { AssetPicker } from "@/components/trading/AssetPicker";
import { SellOptionsResult } from "@/components/trading/SellOptionsResult";
import { SkillsAppliedPanel } from "@/components/trading/SkillsAppliedPanel";
import { findSellOptions } from "@/lib/api-client";
import { SellOptionsRequest } from "@/types/trading";
import { Loader2 } from "lucide-react";

function SellAssetsPageContent() {
  const { isAuthenticated } = useAuth();

  const sellMutation = useMutation({
    mutationFn: (req: SellOptionsRequest) => findSellOptions(req),
  });

  const apiError = sellMutation.isError
    ? sellMutation.error instanceof Error
      ? sellMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Sell from Assets</h1>
        <p className="text-muted-foreground">
          Wähle ein Item aus deinem Besitz und finde heraus, wo du es für das
          meiste Netto-ISK verkaufst — inkl. Route, Sprüngen und Sicherheits-Risiko.
        </p>
      </div>

      {!isAuthenticated && (
        <Alert className="mb-6">
          <AlertTitle>Login erforderlich</AlertTitle>
          <AlertDescription>
            Melde dich per EVE SSO an, um deine Assets zu laden und die besten
            Verkaufsorte zu finden.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-[360px_1fr]">
        {/* Asset picker + sell form */}
        <AssetPicker
          authenticated={isAuthenticated}
          searching={sellMutation.isPending}
          onSearch={(req) => sellMutation.mutate(req)}
        />

        {/* Results */}
        <div className="space-y-6">
          {sellMutation.isPending && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Suche beste Verkaufsorte...
            </div>
          )}

          {apiError && (
            <Alert variant="destructive">
              <AlertTitle>Fehler</AlertTitle>
              <AlertDescription className="flex items-center justify-between gap-4">
                <span>{apiError}</span>
                {sellMutation.variables && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => sellMutation.mutate(sellMutation.variables!)}
                  >
                    Erneut versuchen
                  </Button>
                )}
              </AlertDescription>
            </Alert>
          )}

          {sellMutation.isSuccess && sellMutation.data && (
            <div className="space-y-6">
              <SellOptionsResult
                best={sellMutation.data.best}
                options={sellMutation.data.options}
                notRoutableReason={sellMutation.data.not_routable_reason}
              />
              <SkillsAppliedPanel skills={sellMutation.data.skills_applied} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function SellAssetsPage() {
  return (
    <ErrorBoundary>
      <SellAssetsPageContent />
    </ErrorBoundary>
  );
}

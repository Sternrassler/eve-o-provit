"use client";

import { TradingRoute } from "@/types/trading";
import { TradingRouteCard } from "./TradingRouteCard";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";

interface TradingRouteListProps {
  routes: TradingRoute[];
  loading: boolean;
  error?: string;
  onRetry?: () => void;
}

export function TradingRouteList({
  routes,
  loading,
  error,
  onRetry,
}: TradingRouteListProps) {
  if (loading) {
    return (
      <div className="grid gap-6 md:grid-cols-2">
        {Array.from({ length: 10 }).map((_, i) => (
          <div key={i} className="space-y-3">
            <Skeleton className="h-[300px] w-full rounded-xl" />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="size-4" />
        <AlertTitle>Fehler</AlertTitle>
        <AlertDescription className="flex items-center gap-2">
          <span>{error}</span>
          {onRetry && (
            <Button variant="outline" size="sm" onClick={onRetry}>
              Erneut versuchen
            </Button>
          )}
        </AlertDescription>
      </Alert>
    );
  }

  if (routes.length === 0) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <div className="text-center">
          <p className="text-lg font-medium">Keine Routen gefunden</p>
          <p className="text-sm text-muted-foreground">
            Passe die Filter an oder wähle eine andere Region
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="grid gap-6 md:grid-cols-2">
      {routes.map((route, index) => (
        <TradingRouteCard key={route.item_type_id || route.rank || index} route={route} />
      ))}
    </div>
  );
}

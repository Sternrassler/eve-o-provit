"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { RefreshCw } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

interface ShipRefreshButtonProps {
  characterId: number | null;
  shipTypeId: string;
  authHeader: string | null;
  onRefreshComplete?: () => void;
  disabled?: boolean;
}

export function ShipRefreshButton({
  characterId,
  shipTypeId,
  authHeader,
  onRefreshComplete,
  disabled,
}: ShipRefreshButtonProps) {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const { toast } = useToast();

  const handleRefresh = async () => {
    if (!characterId || !shipTypeId || !authHeader || isRefreshing) return;

    setIsRefreshing(true);
    const startTime = Date.now();

    try {
      toast({
        title: "Fitting-Daten werden aktualisiert",
        description: "Lädt aktuelles Schiffs-Fitting von ESI...",
      });

      // Trigger refresh for ship fitting
      const response = await fetch(
        `${API_BASE_URL}/api/v1/characters/${characterId}/fitting/${shipTypeId}?refresh=true`,
        {
          method: "GET",
          headers: {
            Authorization: authHeader,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
      }

      const duration = ((Date.now() - startTime) / 1000).toFixed(1);

      toast({
        title: "✅ Fitting-Daten aktualisiert",
        description: `Schiff ${shipTypeId} wurde in ${duration}s aktualisiert`,
      });

      onRefreshComplete?.();
    } catch (error) {
      console.error("Failed to refresh fitting data:", error);
      toast({
        title: "❌ Fehler beim Aktualisieren",
        description: error instanceof Error ? error.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    } finally {
      setIsRefreshing(false);
    }
  };

  return (
    <Button
      onClick={handleRefresh}
      disabled={disabled || isRefreshing || !characterId || !shipTypeId || !authHeader}
      size="icon"
      variant="outline"
      className="shrink-0"
      title="Fitting-Daten für dieses Schiff aktualisieren"
    >
      <RefreshCw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
    </Button>
  );
}

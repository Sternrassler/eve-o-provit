"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { RefreshCw } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

interface RegionRefreshButtonProps {
  regionId: string;
  onRefreshComplete?: () => void;
  onRefreshStateChange?: (isRefreshing: boolean) => void;
  disabled?: boolean;
}

export function RegionRefreshButton({
  regionId,
  onRefreshComplete,
  onRefreshStateChange,
  disabled,
}: RegionRefreshButtonProps) {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const { toast } = useToast();

  const handleRefresh = async () => {
    if (!regionId || isRefreshing) return;

    setIsRefreshing(true);
    onRefreshStateChange?.(true);
    const startTime = Date.now();

    try {
      toast({
        title: "Market-Daten werden aktualisiert",
        description: "Dies kann bis zu 60 Sekunden dauern...",
      });

      // Trigger refresh for the entire region (using a dummy typeID)
      // The backend will fetch all market orders for the region
      const response = await fetch(
        `${API_BASE_URL}/api/v1/market/${regionId}/34?refresh=true`,
        {
          method: "GET",
        }
      );

      if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
      }

      const duration = ((Date.now() - startTime) / 1000).toFixed(1);

      toast({
        title: "✅ Market-Daten aktualisiert",
        description: `Region ${regionId} wurde in ${duration}s aktualisiert`,
      });

      onRefreshComplete?.();
    } catch (error) {
      console.error("Failed to refresh market data:", error);
      toast({
        title: "❌ Fehler beim Aktualisieren",
        description: error instanceof Error ? error.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    } finally {
      setIsRefreshing(false);
      onRefreshStateChange?.(false);
    }
  };

  return (
    <Button
      onClick={handleRefresh}
      disabled={disabled || isRefreshing || !regionId}
      size="icon"
      variant="outline"
      className="shrink-0"
      title="Market-Daten für diese Region aktualisieren"
    >
      <RefreshCw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
    </Button>
  );
}

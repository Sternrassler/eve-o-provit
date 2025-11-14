"use client";

import { useEffect, useState } from "react";
import { Clock } from "lucide-react";
import { cn } from "@/lib/utils";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

interface RegionStalenessIndicatorProps {
  regionId: string;
  className?: string;
}

interface MarketDataAge {
  age_minutes: number;
  latest_fetch: string;
  total_orders: number;
}

export function RegionStalenessIndicator({
  regionId,
  className,
}: RegionStalenessIndicatorProps) {
  const [dataAge, setDataAge] = useState<MarketDataAge | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!regionId) return;

    const fetchDataAge = async () => {
      setLoading(true);
      setError(false);

      try {
        const response = await fetch(
          `${API_BASE_URL}/api/v1/market/staleness/${regionId}`
        );

        if (!response.ok) {
          throw new Error("Failed to fetch data age");
        }

        const data = await response.json();
        setDataAge(data);
      } catch (err) {
        console.error("Failed to fetch market data age:", err);
        setError(true);
      } finally {
        setLoading(false);
      }
    };

    fetchDataAge();

    // Refresh indicator every 30 seconds
    const interval = setInterval(fetchDataAge, 30000);
    return () => clearInterval(interval);
  }, [regionId]);

  if (loading || !dataAge) {
    return (
      <div className={cn("flex items-center gap-1.5 text-xs text-muted-foreground", className)}>
        <Clock className="h-3 w-3" />
        <span>Lade...</span>
      </div>
    );
  }

  if (error) {
    return null; // Silent failure
  }

  const { age_minutes } = dataAge;

  // Determine status color
  const getStatusColor = (minutes: number) => {
    if (minutes < 5) return "text-green-600 dark:text-green-400";
    if (minutes < 15) return "text-yellow-600 dark:text-yellow-400";
    return "text-orange-600 dark:text-orange-400";
  };

  // Format time
  const formatAge = (minutes: number) => {
    if (minutes < 1) return "< 1 min";
    if (minutes < 60) return `${Math.floor(minutes)} min`;
    const hours = Math.floor(minutes / 60);
    const mins = Math.floor(minutes % 60);
    return `${hours}h ${mins}m`;
  };

  return (
    <div
      className={cn(
        "flex items-center gap-1.5 text-xs",
        getStatusColor(age_minutes),
        className
      )}
      title={`Letzte Aktualisierung: ${dataAge.latest_fetch}\n${dataAge.total_orders.toLocaleString()} Orders`}
    >
      <Clock className="h-3 w-3" />
      <span>{formatAge(age_minutes)}</span>
    </div>
  );
}

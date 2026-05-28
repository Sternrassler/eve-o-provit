"use client";

import { CompetitionInfo } from "@/types/trading";
import { cn } from "@/lib/utils";
import { Activity } from "lucide-react";

interface CompetitionIndicatorProps {
  competition: CompetitionInfo;
}

/**
 * Shows the order-change frequency for a hub plus a small source badge
 * distinguishing live tracking from a daily baseline estimate.
 */
export function CompetitionIndicator({ competition }: CompetitionIndicatorProps) {
  const isLive = competition.source === "live";
  const label = isLive ? "Live" : "Tages-Baseline";

  return (
    <div className="flex items-center gap-2">
      <Activity className="size-3.5 text-muted-foreground" aria-hidden="true" />
      <span className="text-sm font-medium">
        {competition.changes_per_hour.toFixed(1)}/h
      </span>
      <span
        className={cn(
          "inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-semibold",
          isLive
            ? "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300"
            : "bg-muted text-muted-foreground"
        )}
        title={isLive ? "Live-Order-Tracking" : "Geschätzt aus Tages-Baseline"}
      >
        {label}
      </span>
    </div>
  );
}

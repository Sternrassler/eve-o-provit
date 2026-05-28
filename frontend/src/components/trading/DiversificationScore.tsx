"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { PieChart } from "lucide-react";

interface DiversificationScoreProps {
  score: number; // 0-100
}

function getScoreColor(score: number): string {
  if (score >= 67) return "text-green-600 dark:text-green-400";
  if (score >= 34) return "text-yellow-600 dark:text-yellow-400";
  return "text-red-600 dark:text-red-400";
}

function getScoreLabel(score: number): string {
  if (score >= 67) return "Gut diversifiziert";
  if (score >= 34) return "Mäßig diversifiziert";
  return "Stark konzentriert";
}

/**
 * Diversification score badge (0-100) summarising how well the suggested
 * portfolio spreads capital across items. Higher = better diversified.
 */
export function DiversificationScore({ score }: DiversificationScoreProps) {
  const clamped = Math.max(0, Math.min(100, score));
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="flex items-center gap-2 text-sm font-medium">
          <PieChart
            className="size-4 text-muted-foreground"
            aria-hidden="true"
          />
          Diversifikation
        </CardTitle>
        <Badge variant="secondary">{getScoreLabel(clamped)}</Badge>
      </CardHeader>
      <CardContent>
        <div
          className={cn("text-3xl font-bold", getScoreColor(clamped))}
          data-testid="diversification-score"
        >
          {clamped}
          <span className="ml-1 text-base font-normal text-muted-foreground">
            / 100
          </span>
        </div>
        <div
          className="mt-3 h-2 w-full overflow-hidden rounded-full bg-muted"
          role="progressbar"
          aria-valuenow={clamped}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className={cn(
              "h-full rounded-full transition-all",
              clamped >= 67
                ? "bg-green-600 dark:bg-green-500"
                : clamped >= 34
                  ? "bg-yellow-600 dark:bg-yellow-500"
                  : "bg-red-600 dark:bg-red-500"
            )}
            style={{ width: `${clamped}%` }}
          />
        </div>
      </CardContent>
    </Card>
  );
}

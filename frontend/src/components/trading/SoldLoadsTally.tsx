"use client";

import { cn } from "@/lib/utils";

// Above this many full loads, individual pips become unwieldy → counter + bar.
const PIP_CAP = 24;

interface SoldLoadsTallyProps {
  /** Number of full ore-hold loads the best buy order absorbs (floored, ≥ 1). */
  total: number;
  /** How many loads the user has marked sold (0…total). */
  sold: number;
  onChange: (sold: number) => void;
}

/**
 * A per-ore tally for marking off loads already sold — lets the miner keep
 * track while selling. State is owned by the caller and reset on recalculation.
 */
export function SoldLoadsTally({ total, sold, onChange }: SoldLoadsTallyProps) {
  if (total < 1) return null;
  const clamped = Math.max(0, Math.min(sold, total));

  return (
    <div className="mt-1 space-y-1" data-testid="sold-loads-tally">
      <div className="text-xs text-muted-foreground">
        Verkauft:{" "}
        <span className="font-medium text-foreground">{clamped}</span> / {total}
      </div>
      {total <= PIP_CAP ? (
        <div className="flex flex-wrap gap-1" role="group" aria-label="Verkaufte Ladungen abhaken">
          {Array.from({ length: total }, (_, i) => {
            const filled = i < clamped;
            return (
              <button
                key={i}
                type="button"
                aria-pressed={filled}
                aria-label={`Ladung ${i + 1}${filled ? " — verkauft" : ""}`}
                // Tap fills up to here, or clears from here if already filled.
                onClick={() => onChange(i < clamped ? i : i + 1)}
                className={cn(
                  "h-5 w-5 rounded border text-[11px] leading-none transition-colors",
                  filled
                    ? "border-primary bg-primary/15 text-primary"
                    : "border-muted-foreground/30 text-transparent hover:border-muted-foreground/60",
                )}
              >
                ✓
              </button>
            );
          })}
        </div>
      ) : (
        <div className="flex items-center gap-2">
          <button
            type="button"
            aria-label="Eine Ladung weniger"
            onClick={() => onChange(Math.max(0, clamped - 1))}
            className="h-6 w-6 rounded border border-muted-foreground/30 text-sm leading-none hover:border-muted-foreground/60"
          >
            −
          </button>
          <button
            type="button"
            aria-label="Eine Ladung mehr"
            onClick={() => onChange(Math.min(total, clamped + 1))}
            className="h-6 w-6 rounded border border-muted-foreground/30 text-sm leading-none hover:border-muted-foreground/60"
          >
            +
          </button>
          <div className="h-1.5 flex-1 overflow-hidden rounded bg-muted">
            <div className="h-full bg-primary" style={{ width: `${(clamped / total) * 100}%` }} />
          </div>
        </div>
      )}
    </div>
  );
}

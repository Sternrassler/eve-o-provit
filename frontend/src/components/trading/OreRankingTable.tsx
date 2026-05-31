"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { OreRankRow, SellLocation } from "@/types/trading";
import { cn, formatISK } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface OreRankingTableProps {
  rows: OreRankRow[];
}

function formatSell(loc?: SellLocation): string {
  if (!loc) return "—";
  if (loc.is_structure) return "Player-Structure";
  const parts = [loc.station_name, loc.system_name].filter(Boolean);
  return parts.length ? parts.join(" — ") : "unbekannt";
}

function formatStation(name?: string, system?: string): string {
  const parts = [name, system].filter(Boolean);
  return parts.length ? parts.join(" — ") : "—";
}

/**
 * Ranks ores by ISK/hour (raw sell vs reprocess). Rows arrive pre-sorted by
 * best ISK/hour descending. Each row expands to show where to reprocess and
 * where to sell (raw ore, or a per-mineral breakdown for the refine path).
 */
export function OreRankingTable({ rows }: OreRankingTableProps) {
  if (rows.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Erz-Ranking</CardTitle>
        </CardHeader>
        <CardContent>
          <div
            data-testid="ore-ranking-empty"
            className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
          >
            Keine Erz-Daten für diese Region und Sicherheitsklasse gefunden
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Erz-Ranking</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto">
          <table className="w-full text-sm" aria-label="Erz-Ranking">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th scope="col" className="px-4 py-3 font-medium">Erz</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">m³/h</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">ISK/h roh</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">ISK/h refine</th>
                <th scope="col" className="px-4 py-3 font-medium">Verdict</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Steuer</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Δ ISK/h</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <OreRankRow key={row.ore_type_id} row={row} />
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

function OreRankRow({ row }: { row: OreRankRow }) {
  const [open, setOpen] = useState(false);
  const isRefine = row.best === "refine";
  const verdictText = isRefine ? "Reprozessieren" : "Roh verkaufen";
  const verdictClass = isRefine
    ? "text-blue-600 dark:text-blue-400 font-medium"
    : "text-green-600 dark:text-green-400 font-medium";

  const m3PerHour =
    row.mining_m3_per_hour >= 1000
      ? `${(row.mining_m3_per_hour / 1000).toFixed(1)}k`
      : String(row.mining_m3_per_hour);

  return (
    <>
      <tr
        data-testid="ore-ranking-row"
        data-ore-type-id={row.ore_type_id}
        className="cursor-pointer border-b transition-colors hover:bg-muted/40"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <td className="px-4 py-3 font-medium">
          <span className="inline-flex items-center gap-1">
            {open ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
            {row.ore_name}
          </span>
        </td>
        <td className="px-4 py-3 text-right">{m3PerHour}</td>
        <td className="px-4 py-3 text-right">{formatISK(row.raw_isk_per_hour)}</td>
        <td className="px-4 py-3 text-right">{formatISK(row.refine_isk_per_hour)}</td>
        <td className={cn("px-4 py-3", verdictClass)}>{verdictText}</td>
        <td className="px-4 py-3 text-right">{(row.best_station_tax * 100).toFixed(1)}%</td>
        <td className="px-4 py-3 text-right font-medium text-green-600 dark:text-green-400">
          {formatISK(row.delta_isk_per_hour)}
        </td>
      </tr>
      {open && (
        <tr data-testid="ore-ranking-detail" className="border-b bg-muted/20">
          <td colSpan={7} className="px-4 py-3">
            {isRefine ? (
              <div className="space-y-2">
                <div className="text-sm">
                  <span className="text-muted-foreground">Aufbereiten bei: </span>
                  {formatStation(row.best_station_name, row.best_station_system)}
                </div>
                <div className="text-sm text-muted-foreground">Raffinate verkaufen:</div>
                <table className="w-full text-xs">
                  <thead>
                    <tr className="text-left text-muted-foreground">
                      <th className="py-1 pr-4 font-medium">Mineral</th>
                      <th className="py-1 pr-4 text-right font-medium">Menge</th>
                      <th className="py-1 pr-4 text-right font-medium">Buy</th>
                      <th className="py-1 font-medium">Verkaufsort</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(row.materials ?? []).map((m) => (
                      <tr key={m.material_type_id}>
                        <td className="py-1 pr-4">{m.material_name || `Typ ${m.material_type_id}`}</td>
                        <td className="py-1 pr-4 text-right">{m.effective_qty.toLocaleString("de-DE")}</td>
                        <td className="py-1 pr-4 text-right">{formatISK(m.buy_price)}</td>
                        <td className="py-1">{formatSell(m.sell)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="text-sm">
                <span className="text-muted-foreground">Roh verkaufen bei: </span>
                {formatSell(row.raw_sell)}
              </div>
            )}
          </td>
        </tr>
      )}
    </>
  );
}

"use client";

import type { OreRankRow } from "@/types/trading";
import { cn, formatISK } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface OreRankingTableProps {
  rows: OreRankRow[];
}

/**
 * Ranks ores by ISK/hour (raw sell vs reprocess). Rows arrive pre-sorted by
 * best ISK/hour descending. An empty rows array means no ore data was found
 * for the given region/sec-band combination.
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
                <th scope="col" className="px-4 py-3 font-medium">
                  Erz
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  m³/h
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  ISK/h roh
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  ISK/h refine
                </th>
                <th scope="col" className="px-4 py-3 font-medium">
                  Verdict
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  Steuer
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  Δ ISK/h
                </th>
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
    <tr
      data-testid="ore-ranking-row"
      data-ore-type-id={row.ore_type_id}
      className="border-b transition-colors hover:bg-muted/40"
    >
      <td className="px-4 py-3 font-medium">{row.ore_name}</td>
      <td className="px-4 py-3 text-right">{m3PerHour}</td>
      <td className="px-4 py-3 text-right">{formatISK(row.raw_isk_per_hour)}</td>
      <td className="px-4 py-3 text-right">{formatISK(row.refine_isk_per_hour)}</td>
      <td className={cn("px-4 py-3", verdictClass)}>{verdictText}</td>
      <td className="px-4 py-3 text-right">
        {(row.best_station_tax * 100).toFixed(1)}%
      </td>
      <td className="px-4 py-3 text-right font-medium text-green-600 dark:text-green-400">
        {formatISK(row.delta_isk_per_hour)}
      </td>
    </tr>
  );
}

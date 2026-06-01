"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { OreRankRow, SellLocation } from "@/types/trading";
import { cn, formatISK } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/hooks/use-toast";
import { openMarketDetails, setWaypoint } from "@/lib/api-client";

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
                <th scope="col" className="px-4 py-3 text-right font-medium">ISK/h eff.</th>
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
  const { toast } = useToast();
  const linkCls = "text-left hover:underline underline-offset-2 hover:text-primary transition-colors";
  const openMarket = async (typeId: number, name: string) => {
    try {
      await openMarketDetails(typeId);
      toast({
        title: "Markt-Detail an EVE gesendet",
        description: `${name} — falls nichts passiert: Markt-Fenster im Spiel (Alt+R) öffnen und nochmal klicken.`,
      });
    } catch (err) {
      toast({
        title: "Fehler",
        description: err instanceof Error ? err.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    }
  };
  const setRoute = async (destId: number, label: string) => {
    try {
      await setWaypoint(destId, { clearOtherWaypoints: true });
      toast({ title: "Route gesetzt", description: `Waypoint in EVE gesetzt: ${label}` });
    } catch (err) {
      toast({
        title: "Fehler",
        description: err instanceof Error ? err.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    }
  };
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
            <button
              type="button"
              className={linkCls}
              title="Marktdetails im EVE-Client öffnen"
              onClick={(e) => {
                e.stopPropagation();
                openMarket(row.ore_type_id, row.ore_name);
              }}
            >
              {row.ore_name}
            </button>
            {row.is_estimate && (
              <span
                data-testid="ore-estimate-badge"
                title={row.estimate_reason || "Schätzwert"}
                className="ml-2 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-800 dark:bg-amber-900/30 dark:text-amber-200"
              >
                ≈ Schätzung
              </span>
            )}
          </span>
        </td>
        <td className="px-4 py-3 text-right">{m3PerHour}</td>
        <td className="px-4 py-3 text-right">{formatISK(row.raw_isk_per_hour)}</td>
        <td className="px-4 py-3 text-right">{formatISK(row.refine_isk_per_hour)}</td>
        <td className={cn("px-4 py-3", verdictClass)}>{verdictText}</td>
        <td className="px-4 py-3 text-right">{(row.best_station_tax * 100).toFixed(1)}%</td>
        <td data-testid="ore-effective-isk" className="px-4 py-3 text-right font-medium text-green-600 dark:text-green-400">
          {formatISK(row.effective_isk_per_hour ?? row.delta_isk_per_hour)}
        </td>
      </tr>
      {open && (
        <tr data-testid="ore-ranking-detail" className="border-b bg-muted/20">
          <td colSpan={7} className="px-4 py-3">
            {isRefine ? (
              <div className="space-y-2">
                {row.cycle_minutes != null && (
                  <div className="mb-2 text-xs text-muted-foreground">
                    Zyklus {row.cycle_minutes.toFixed(1)} min · {row.route_jumps ?? 0} Jumps
                    {row.sell_system_name ? ` · Verkauf in ${row.sell_system_name}` : ""}
                  </div>
                )}
                <div className="text-sm">
                  <span className="text-muted-foreground">Aufbereiten bei: </span>
                  {row.best_station_id ? (
                    <button
                      type="button"
                      className={linkCls}
                      title="Route in EVE setzen"
                      onClick={() =>
                        setRoute(
                          row.best_station_id!,
                          formatStation(row.best_station_name, row.best_station_system)
                        )
                      }
                    >
                      {formatStation(row.best_station_name, row.best_station_system)}
                    </button>
                  ) : (
                    formatStation(row.best_station_name, row.best_station_system)
                  )}
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
                        <td className="py-1 pr-4">
                          <button
                            type="button"
                            className={linkCls}
                            title="Marktdetails im EVE-Client öffnen"
                            onClick={() =>
                              openMarket(m.material_type_id, m.material_name || `Typ ${m.material_type_id}`)
                            }
                          >
                            {m.material_name || `Typ ${m.material_type_id}`}
                          </button>
                        </td>
                        <td className="py-1 pr-4 text-right">{m.effective_qty.toLocaleString("de-DE")}</td>
                        <td className="py-1 pr-4 text-right">{formatISK(m.buy_price)}</td>
                        <td className="py-1">
                          {m.sell.location_id ? (
                            <button
                              type="button"
                              className={linkCls}
                              title="Route in EVE setzen"
                              onClick={() => setRoute(m.sell.location_id!, formatSell(m.sell))}
                            >
                              {formatSell(m.sell)}
                            </button>
                          ) : (
                            formatSell(m.sell)
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="text-sm">
                {row.cycle_minutes != null && (
                  <div className="mb-2 text-xs text-muted-foreground">
                    Zyklus {row.cycle_minutes.toFixed(1)} min · {row.route_jumps ?? 0} Jumps
                    {row.sell_system_name ? ` · Verkauf in ${row.sell_system_name}` : ""}
                  </div>
                )}
                <span className="text-muted-foreground">Roh verkaufen bei: </span>
                {row.raw_sell?.location_id ? (
                  <button
                    type="button"
                    className={linkCls}
                    title="Route in EVE setzen"
                    onClick={() => setRoute(row.raw_sell!.location_id!, formatSell(row.raw_sell))}
                  >
                    {formatSell(row.raw_sell)}
                  </button>
                ) : (
                  formatSell(row.raw_sell)
                )}
              </div>
            )}
          </td>
        </tr>
      )}
    </>
  );
}

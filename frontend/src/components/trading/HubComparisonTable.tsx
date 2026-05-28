"use client";

import { HubComparisonResult, HubRow } from "@/types/trading";
import {
  cn,
  formatISKWithSeparators,
  getSpreadColor,
  getProfitColor,
} from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RecommendedHubBadge } from "./RecommendedHubBadge";
import { CompetitionIndicator } from "./CompetitionIndicator";

interface HubComparisonTableProps {
  result: HubComparisonResult;
}

/**
 * Side-by-side comparison of an item's station-trading profitability across
 * the major hubs. Rows arrive pre-sorted by net margin descending, with
 * no-data hubs last. The recommended (best) hub is highlighted.
 */
export function HubComparisonTable({ result }: HubComparisonTableProps) {
  const hasData = result.hubs.some((h) => h.has_data);
  const noProfitableHub = hasData && !result.best_hub_region_id;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">
          Hub-Vergleich: {result.item_name}
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {noProfitableHub && (
          <div
            data-testid="no-profit-hint"
            className="mx-4 mt-4 rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
          >
            Kein profitabler Hub für dieses Item — nach Steuer und Broker-Fee ist die
            Station-Trading-Marge an allen Hubs negativ. Der beste (am wenigsten
            verlustreiche) Hub steht oben.
          </div>
        )}
        <div className="overflow-x-auto">
          <table className="w-full text-sm" aria-label="Hub-Vergleich">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th scope="col" className="px-4 py-3 font-medium">Hub</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Kaufpreis</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Verkaufspreis</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Spread</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Netto-Marge</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Netto/Einheit</th>
                <th scope="col" className="px-4 py-3 text-right font-medium">Tagesvolumen</th>
                <th scope="col" className="px-4 py-3 font-medium">Konkurrenz</th>
              </tr>
            </thead>
            <tbody>
              {result.hubs.map((hub) => (
                <HubRowItem
                  key={hub.region_id}
                  hub={hub}
                  isRecommended={hub.region_id === result.best_hub_region_id}
                />
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

function HubRowItem({
  hub,
  isRecommended,
}: {
  hub: HubRow;
  isRecommended: boolean;
}) {
  if (!hub.has_data) {
    return (
      <tr
        data-testid="hub-row"
        data-region-id={hub.region_id}
        className="border-b text-muted-foreground"
      >
        <td className="px-4 py-3">
          <div className="font-medium">{hub.hub_name}</div>
          <div className="text-xs">{hub.region_name}</div>
        </td>
        <td className="px-4 py-3 text-center italic" colSpan={7}>
          Keine Daten verfügbar
        </td>
      </tr>
    );
  }

  return (
    <tr
      data-testid="hub-row"
      data-region-id={hub.region_id}
      className={cn(
        "border-b transition-colors hover:bg-muted/40",
        isRecommended && "bg-green-50 dark:bg-green-900/20"
      )}
    >
      <td className="px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="font-medium">{hub.hub_name}</span>
          {isRecommended && <RecommendedHubBadge />}
        </div>
        <div className="text-xs text-muted-foreground">
          {hub.region_name} · {hub.tier}
        </div>
      </td>
      <td className="px-4 py-3 text-right font-medium">
        {formatISKWithSeparators(hub.buy_price)}
      </td>
      <td className="px-4 py-3 text-right font-medium">
        {formatISKWithSeparators(hub.sell_price)}
      </td>
      <td
        className={cn(
          "px-4 py-3 text-right font-medium",
          getSpreadColor(hub.spread_percent)
        )}
      >
        {hub.spread_percent.toFixed(1)}%
      </td>
      <td
        className={cn(
          "px-4 py-3 text-right font-bold",
          getProfitColor(hub.net_margin_percent)
        )}
      >
        {hub.net_margin_percent.toFixed(1)}%
      </td>
      <td className="px-4 py-3 text-right font-medium">
        {formatISKWithSeparators(hub.net_profit_per_unit)}
      </td>
      <td className="px-4 py-3 text-right">
        {hub.daily_volume.toLocaleString("de-DE")}
      </td>
      <td className="px-4 py-3">
        <CompetitionIndicator competition={hub.competition} />
      </td>
    </tr>
  );
}

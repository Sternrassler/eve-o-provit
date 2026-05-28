"use client";

import { PortfolioResult, PortfolioItem } from "@/types/trading";
import { cn, formatISKWithSeparators, getProfitColor } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface PortfolioResultTableProps {
  result: PortfolioResult;
}

/**
 * Suggested capital allocation across items (the portfolio), maximising daily
 * profit. Rows arrive pre-sorted by efficiency (most efficient first). An empty
 * item list means no viable portfolio for the given budget.
 */
export function PortfolioResultTable({ result }: PortfolioResultTableProps) {
  if (result.items.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Portfolio</CardTitle>
        </CardHeader>
        <CardContent>
          <div
            data-testid="portfolio-empty"
            className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
          >
            Kein sinnvolles Portfolio für dieses Budget
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Kapital-Allokation</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto">
          <table className="w-full text-sm" aria-label="Kapital-Allokation">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th scope="col" className="px-4 py-3 font-medium">
                  Item
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  Kapital
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  Units
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  Fahrten/Tag
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  Tagesgewinn
                </th>
                <th scope="col" className="px-4 py-3 text-right font-medium">
                  ROI%
                </th>
              </tr>
            </thead>
            <tbody>
              {result.items.map((item) => (
                <PortfolioRowItem key={item.type_id} item={item} />
              ))}
            </tbody>
            <tfoot>
              <tr
                data-testid="portfolio-totals"
                className="border-t-2 font-semibold"
              >
                <td className="px-4 py-3">Gesamt</td>
                <td className="px-4 py-3 text-right">
                  {formatISKWithSeparators(result.total_capital_used)}
                </td>
                <td className="px-4 py-3" colSpan={2} />
                <td className="px-4 py-3 text-right text-green-600 dark:text-green-400">
                  {formatISKWithSeparators(result.total_daily_profit)}
                </td>
                <td className="px-4 py-3" />
              </tr>
            </tfoot>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

function PortfolioRowItem({ item }: { item: PortfolioItem }) {
  return (
    <tr
      data-testid="portfolio-row"
      data-type-id={item.type_id}
      className="border-b transition-colors hover:bg-muted/40"
    >
      <td className="px-4 py-3 font-medium">{item.name}</td>
      <td className="px-4 py-3 text-right">
        {formatISKWithSeparators(item.capital_used)}
      </td>
      <td className="px-4 py-3 text-right">
        {item.units.toLocaleString("de-DE")}
      </td>
      <td className="px-4 py-3 text-right">{item.trips_per_day}</td>
      <td className="px-4 py-3 text-right font-medium text-green-600 dark:text-green-400">
        {formatISKWithSeparators(item.daily_profit)}
      </td>
      <td
        className={cn(
          "px-4 py-3 text-right font-bold",
          getProfitColor(item.roi_percent)
        )}
      >
        {item.roi_percent.toFixed(1)}%
      </td>
    </tr>
  );
}

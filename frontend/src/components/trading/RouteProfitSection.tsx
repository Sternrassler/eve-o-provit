"use client";

import { TradingRoute } from "@/types/trading";
import { cn, formatISK, formatISKWithSeparators, getSpreadColor, getProfitColor } from "@/lib/utils";
import { Info } from "lucide-react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { FeeBreakdown } from "./FeeBreakdown";

interface RouteProfitSectionProps {
  route: TradingRoute;
}

export function RouteProfitSection({ route }: RouteProfitSectionProps) {
  const isMultiTour = route.number_of_tours && route.number_of_tours > 1;

  return (
    <>
      {/* Quantity */}
      <div className="text-sm text-muted-foreground">
        Menge: {route.quantity.toLocaleString("de-DE")}
        {route.item_volume && (
          <span className="text-xs ml-1">
            ({route.item_volume} m³/Einheit,{" "}
            {(route.quantity * route.item_volume).toFixed(0)} m³ gesamt)
          </span>
        )}
        {isMultiTour && route.number_of_tours && (
          <span className="block sm:inline">
            {" "}
            (≈{Math.ceil(route.quantity / route.number_of_tours).toLocaleString("de-DE")}{" "}
            pro Tour)
          </span>
        )}
      </div>

      {/* Prices */}
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <div className="text-muted-foreground">Kaufpreis</div>
          <div className="font-medium">{formatISK(route.buy_price)}</div>
        </div>
        <div>
          <div className="text-muted-foreground">Verkaufspreis</div>
          <div className="font-medium">{formatISK(route.sell_price)}</div>
        </div>
      </div>

      {/* Fee Breakdown and Profit Display */}
      {route.gross_profit !== undefined && route.net_profit !== undefined ? (
        <div className="space-y-3 border-t pt-3">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-sm text-muted-foreground">Brutto-Gewinn</div>
              <div className="font-medium">
                {formatISKWithSeparators(route.gross_profit)}
              </div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Spread</div>
              <div className={cn("font-medium", getSpreadColor(route.spread_percent))}>
                {route.spread_percent.toFixed(1)}%
              </div>
            </div>
          </div>

          {route.total_fees !== undefined && (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="flex items-center justify-between cursor-help border-t pt-3">
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm text-muted-foreground">Gebühren</span>
                      <Info className="size-3.5 text-muted-foreground" />
                    </div>
                    <span className="text-sm font-medium text-red-600 dark:text-red-400">
                      -{formatISKWithSeparators(route.total_fees)}
                    </span>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="top" className="max-w-xs">
                  <FeeBreakdown
                    fees={{
                      salesTax: route.sales_tax || 0,
                      brokerFees: route.broker_fees || 0,
                      estimatedRelistFee: route.estimated_relist_fee || 0,
                      totalFees: route.total_fees,
                    }}
                  />
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}

          <div className="grid grid-cols-2 gap-4 border-t pt-3">
            <div>
              <div className="text-sm text-muted-foreground">Netto-Gewinn</div>
              <div
                className={cn(
                  "text-lg font-bold",
                  getProfitColor(route.net_profit_percent || 0)
                )}
              >
                {formatISKWithSeparators(route.net_profit)}
              </div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Netto-Marge</div>
              <div
                className={cn(
                  "text-lg font-bold",
                  getProfitColor(route.net_profit_percent || 0)
                )}
              >
                {(route.net_profit_percent || 0).toFixed(1)}%
              </div>
            </div>
          </div>
        </div>
      ) : (
        /* Fallback when fee data is not available */
        <>
          {isMultiTour ? (
            <div className="space-y-2 border-t pt-3">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-sm text-muted-foreground">Pro Tour</div>
                  <div className="font-bold">
                    {formatISK(route.profit_per_tour || 0)}
                  </div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">Gesamt</div>
                  <div className="text-lg font-bold">
                    {formatISK(route.total_profit || route.profit || 0)}
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-4 border-t pt-3">
              <div>
                <div className="text-sm text-muted-foreground">Gewinn</div>
                <div className="text-lg font-bold">
                  {formatISK(route.total_profit || route.profit || 0)}
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Spread</div>
                <div className={cn("text-lg font-bold", getSpreadColor(route.spread_percent))}>
                  {route.spread_percent.toFixed(1)}%
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* Spread for multi-tour (shown below profit) */}
      {isMultiTour && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Spread</span>
          <span className={cn("font-bold", getSpreadColor(route.spread_percent))}>
            {route.spread_percent.toFixed(1)}%
          </span>
        </div>
      )}
    </>
  );
}

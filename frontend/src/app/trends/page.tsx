"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { TrendingUp, TrendingDown, Minus } from "lucide-react";

export default function TrendsPage() {
  return (
    <div className="container mx-auto p-8">
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl font-bold">Price Trend Analysis</h1>
          <Badge variant="outline" className="text-purple-600">Phase 3</Badge>
        </div>
        <p className="text-muted-foreground">
          Analysiere Preistrends und erkenne Spekulationsgelegenheiten
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-3 mb-8">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Bullish Trends
            </CardTitle>
            <TrendingUp className="h-4 w-4 text-green-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">↗ Upward</div>
            <p className="text-xs text-muted-foreground">
              30/60/90 day analysis
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Sideways
            </CardTitle>
            <Minus className="h-4 w-4 text-yellow-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-600">→ Stable</div>
            <p className="text-xs text-muted-foreground">
              Low volatility
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Bearish Trends
            </CardTitle>
            <TrendingDown className="h-4 w-4 text-red-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">↘ Downward</div>
            <p className="text-xs text-muted-foreground">
              Avoid these items
            </p>
          </CardContent>
        </Card>
      </div>

      <Card className="border-dashed">
        <CardHeader>
          <CardTitle>Coming in Phase 3</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">📈</span>
              Historical Price Charts
            </h3>
            <p className="text-sm text-muted-foreground">
              View 30/60/90 day price history with trend lines and moving averages
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">🎯</span>
              Buy/Sell Signals
            </h3>
            <p className="text-sm text-muted-foreground">
              Automated detection of optimal entry and exit points
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">📊</span>
              Volatility Metrics
            </h3>
            <p className="text-sm text-muted-foreground">
              Understand price stability and risk for each item
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">🔮</span>
              Speculation Support
            </h3>
            <p className="text-sm text-muted-foreground">
              Predict patch impacts and seasonal market changes
            </p>
          </div>

          <div className="mt-6 p-4 bg-muted rounded-lg">
            <p className="text-sm">
              <strong>Target Release:</strong> June 2026 (Phase 3)
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Dependencies: Historical Data Collection (new feature)
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

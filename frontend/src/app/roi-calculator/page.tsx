"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Calculator, PieChart, Wallet } from "lucide-react";

export default function ROICalculatorPage() {
  return (
    <div className="container mx-auto p-8">
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl font-bold">ROI Calculator & Capital Optimizer</h1>
          <Badge variant="outline" className="text-blue-600">Phase 2</Badge>
        </div>
        <p className="text-muted-foreground">
          Optimiere deine Kapital-Allokation für maximalen Return on Investment
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-3 mb-8">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              ROI Ranking
            </CardTitle>
            <Calculator className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Daily Profit / Capital</div>
            <p className="text-xs text-muted-foreground">
              Sort by efficiency
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Portfolio Builder
            </CardTitle>
            <PieChart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Auto-Allocate</div>
            <p className="text-xs text-muted-foreground">
              "Invest 500M optimally"
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Risk Diversification
            </CardTitle>
            <Wallet className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Score</div>
            <p className="text-xs text-muted-foreground">
              Balance your portfolio
            </p>
          </CardContent>
        </Card>
      </div>

      <Card className="border-dashed">
        <CardHeader>
          <CardTitle>Coming in Phase 2</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">📈</span>
              Smart Capital Allocation
            </h3>
            <p className="text-sm text-muted-foreground">
              Enter your available capital and get optimal item mix for maximum daily profit
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">🎲</span>
              Risk Management
            </h3>
            <p className="text-sm text-muted-foreground">
              Diversification score prevents over-concentration in single items
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">💰</span>
              Expected Profit Projection
            </h3>
            <p className="text-sm text-muted-foreground">
              See projected daily/weekly/monthly profits based on historical data
            </p>
          </div>

          <div className="mt-6 p-4 bg-muted rounded-lg">
            <p className="text-sm">
              <strong>Target Release:</strong> March 2026 (Phase 2)
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Dependencies: Volume Filter (#42), Fee Calculator (#38)
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

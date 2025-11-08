"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Building2, TrendingUp, Users } from "lucide-react";

export default function MultiHubPage() {
  return (
    <div className="container mx-auto p-8">
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl font-bold">Multi-Hub Comparison</h1>
          <Badge variant="outline" className="text-blue-600">Phase 2</Badge>
        </div>
        <p className="text-muted-foreground">
          Vergleiche Margen über verschiedene Trading Hubs hinweg
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-3 mb-8">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Trade Hubs
            </CardTitle>
            <Building2 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">4+</div>
            <p className="text-xs text-muted-foreground">
              Jita, Dodixie, Amarr, Rens
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Competition Level
            </CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Low/Med/High</div>
            <p className="text-xs text-muted-foreground">
              Order update frequency tracking
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Best Hub Recommendation
            </CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Auto</div>
            <p className="text-xs text-muted-foreground">
              Based on your capital
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
              <span className="text-2xl">📊</span>
              Side-by-Side Hub Comparison
            </h3>
            <p className="text-sm text-muted-foreground">
              Compare margins, volumes, and competition across all major trade hubs
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">🎯</span>
              Smart Hub Recommendation
            </h3>
            <p className="text-sm text-muted-foreground">
              Get personalized recommendations based on your available capital
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">⚠️</span>
              Competition Indicator
            </h3>
            <p className="text-sm text-muted-foreground">
              Avoid high-competition items (0.01 ISK wars) with real-time tracking
            </p>
          </div>

          <div className="mt-6 p-4 bg-muted rounded-lg">
            <p className="text-sm">
              <strong>Target Release:</strong> March 2026 (Phase 2)
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Dependencies: Fee Calculator (#38), Volume Filter (#42)
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

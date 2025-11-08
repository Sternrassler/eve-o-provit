"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Bell, Star, Mail } from "lucide-react";

export default function WatchlistPage() {
  return (
    <div className="container mx-auto p-8">
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl font-bold">Watchlist & Alerts</h1>
          <Badge variant="outline" className="text-purple-600">Phase 3</Badge>
        </div>
        <p className="text-muted-foreground">
          Beobachte Items und erhalte Benachrichtigungen bei Profit-Gelegenheiten
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-3 mb-8">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Custom Watchlists
            </CardTitle>
            <Star className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Unlimited</div>
            <p className="text-xs text-muted-foreground">
              Track your favorite items
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Price Alerts
            </CardTitle>
            <Bell className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Real-Time</div>
            <p className="text-xs text-muted-foreground">
              Browser notifications
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Email Alerts
            </CardTitle>
            <Mail className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Optional</div>
            <p className="text-xs text-muted-foreground">
              Daily digest available
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
              <span className="text-2xl">⭐</span>
              Personal Watchlists
            </h3>
            <p className="text-sm text-muted-foreground">
              Create unlimited watchlists to track different trading strategies
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">🔔</span>
              Configurable Alerts
            </h3>
            <p className="text-sm text-muted-foreground">
              Set custom thresholds for price, margin, and volume changes
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">📱</span>
              Multi-Channel Notifications
            </h3>
            <p className="text-sm text-muted-foreground">
              Browser push notifications + optional email digest
            </p>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <span className="text-2xl">🤖</span>
              Smart Alerts
            </h3>
            <p className="text-sm text-muted-foreground">
              AI-powered suggestions based on your trading history
            </p>
          </div>

          <div className="mt-6 p-4 bg-muted rounded-lg">
            <p className="text-sm">
              <strong>Target Release:</strong> June 2026 (Phase 3)
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Dependencies: Trend Analysis (#47), User Preferences System
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

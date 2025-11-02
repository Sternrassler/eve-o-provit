"use client";

import { useState } from "react";
import { TradingFilters as TradingFiltersType } from "@/types/trading";
import { Slider } from "@/components/ui/slider";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ChevronDown, ChevronUp, RotateCcw } from "lucide-react";

interface TradingFiltersProps {
  filters: TradingFiltersType;
  onChange: (filters: TradingFiltersType) => void;
}

const defaultFilters: TradingFiltersType = {
  minSpread: 5,
  minProfit: 100000,
  maxTravelTime: 30,
  allowHighSec: true,
  allowLowSec: false,
  allowNullSec: false,
};

export function TradingFilters({ filters, onChange }: TradingFiltersProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const handleReset = () => {
    onChange(defaultFilters);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>Filter</span>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleReset}
              className="gap-1"
            >
              <RotateCcw className="size-4" />
              Zurücksetzen
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsCollapsed(!isCollapsed)}
            >
              {isCollapsed ? (
                <ChevronDown className="size-4" />
              ) : (
                <ChevronUp className="size-4" />
              )}
            </Button>
          </div>
        </CardTitle>
      </CardHeader>
      {!isCollapsed && (
        <CardContent className="space-y-6">
          {/* Min Spread */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Min. Spread</label>
              <span className="text-sm text-muted-foreground">
                {filters.minSpread}%
              </span>
            </div>
            <Slider
              value={[filters.minSpread]}
              onValueChange={(value) =>
                onChange({ ...filters, minSpread: value[0] })
              }
              min={0}
              max={50}
              step={1}
            />
          </div>

          {/* Min Profit */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Min. Gewinn (ISK)</label>
            <Input
              type="number"
              value={filters.minProfit}
              onChange={(e) =>
                onChange({ ...filters, minProfit: Number(e.target.value) })
              }
              min={0}
              step={10000}
            />
          </div>

          {/* Max Travel Time */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Max. Reisezeit</label>
              <span className="text-sm text-muted-foreground">
                {filters.maxTravelTime}min
              </span>
            </div>
            <Slider
              value={[filters.maxTravelTime]}
              onValueChange={(value) =>
                onChange({ ...filters, maxTravelTime: value[0] })
              }
              min={0}
              max={60}
              step={5}
            />
          </div>

          {/* Security Zones */}
          <div className="space-y-3">
            <label className="text-sm font-medium">Sicherheitszonen</label>
            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="high-sec"
                  checked={filters.allowHighSec}
                  onCheckedChange={(checked) =>
                    onChange({ ...filters, allowHighSec: checked === true })
                  }
                />
                <label
                  htmlFor="high-sec"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer"
                >
                  High Sec (≥0.5)
                </label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="low-sec"
                  checked={filters.allowLowSec}
                  onCheckedChange={(checked) =>
                    onChange({ ...filters, allowLowSec: checked === true })
                  }
                />
                <label
                  htmlFor="low-sec"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer"
                >
                  Low Sec (0.1-0.4)
                </label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="null-sec"
                  checked={filters.allowNullSec}
                  onCheckedChange={(checked) =>
                    onChange({ ...filters, allowNullSec: checked === true })
                  }
                />
                <label
                  htmlFor="null-sec"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer"
                >
                  Null Sec (≤0.0)
                </label>
              </div>
            </div>
          </div>
        </CardContent>
      )}
    </Card>
  );
}

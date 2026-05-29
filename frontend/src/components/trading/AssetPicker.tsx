"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowDownAZ, ArrowUpAZ, Loader2, Search } from "lucide-react";
import { AssetItem, SellOptionsRequest } from "@/types/trading";
import { listAssets } from "@/lib/api-client";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";

interface AssetPickerProps {
  authenticated: boolean;
  /** Disable while a sell-options search is in flight. */
  searching?: boolean;
  /** Fired with the assembled sell-options request when the user searches. */
  onSearch: (req: SellOptionsRequest) => void;
}

/**
 * Picker for the character's marketable assets. Fetches the asset list, lets
 * the user filter by name and pick a stack, then choose a quantity + low-sec
 * preference and fire a sell-options search. Non-marketable assets are shown
 * but disabled (no market group).
 */
export function AssetPicker({
  authenticated,
  searching = false,
  onSearch,
}: AssetPickerProps) {
  const [filter, setFilter] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "quantity">("name");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [selected, setSelected] = useState<AssetItem | null>(null);
  const [quantity, setQuantity] = useState<number>(0);
  const [avoidLowSec, setAvoidLowSec] = useState(true);

  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["assets"],
    queryFn: listAssets,
    enabled: authenticated,
  });

  const filtered = useMemo(() => {
    const assets = data?.assets ?? [];
    const q = filter.trim().toLowerCase();
    const matched = q
      ? assets.filter((a) => a.name.toLowerCase().includes(q))
      : [...assets];
    const dir = sortDir === "asc" ? 1 : -1;
    matched.sort((a, b) => {
      const cmp =
        sortBy === "name"
          ? a.name.localeCompare(b.name, "de")
          : a.quantity - b.quantity;
      return cmp * dir;
    });
    return matched;
  }, [data, filter, sortBy, sortDir]);

  const handleSelect = (asset: AssetItem) => {
    if (!asset.marketable) return;
    setSelected(asset);
    setQuantity(asset.quantity);
  };

  const handleSubmit = () => {
    if (!selected) return;
    onSearch({
      type_id: selected.type_id,
      location_id: selected.location_id,
      quantity,
      avoid_low_sec: avoidLowSec,
    });
  };

  if (!authenticated) {
    return null;
  }

  return (
    <div className="space-y-4 rounded-lg border p-4" data-testid="asset-picker">
      <div className="space-y-2">
        <Label htmlFor="asset-filter">Asset suchen</Label>
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
          <Input
            id="asset-filter"
            placeholder="Nach Name filtern..."
            value={filter}
            disabled={isLoading}
            className="pl-8"
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
      </div>

      {!isLoading && !isError && (data?.assets?.length ?? 0) > 0 && (
        <div className="flex items-center gap-2" data-testid="asset-sort">
          <span className="text-xs text-muted-foreground">Sortieren:</span>
          <Button
            type="button"
            variant={sortBy === "name" ? "secondary" : "ghost"}
            size="sm"
            data-testid="sort-name"
            onClick={() => setSortBy("name")}
          >
            Name
          </Button>
          <Button
            type="button"
            variant={sortBy === "quantity" ? "secondary" : "ghost"}
            size="sm"
            data-testid="sort-quantity"
            onClick={() => setSortBy("quantity")}
          >
            Menge
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            data-testid="sort-dir"
            aria-label={sortDir === "asc" ? "Aufsteigend" : "Absteigend"}
            onClick={() => setSortDir((d) => (d === "asc" ? "desc" : "asc"))}
          >
            {sortDir === "asc" ? (
              <ArrowDownAZ className="size-4" />
            ) : (
              <ArrowUpAZ className="size-4" />
            )}
          </Button>
        </div>
      )}

      {isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Lade Assets...
        </div>
      )}

      {isError && (
        <Alert variant="destructive">
          <AlertTitle>Fehler</AlertTitle>
          <AlertDescription className="flex items-center justify-between gap-4">
            <span>
              {error instanceof Error ? error.message : "Unbekannter Fehler"}
            </span>
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              Erneut versuchen
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {!isLoading && !isError && filtered.length === 0 && (
        <div
          data-testid="asset-picker-empty"
          className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
        >
          Keine Assets gefunden
        </div>
      )}

      {filtered.length > 0 && (
        <ul
          className="max-h-72 space-y-1 overflow-y-auto"
          data-testid="asset-list"
        >
          {filtered.map((asset) => {
            const isSelected =
              selected?.type_id === asset.type_id &&
              selected?.location_id === asset.location_id;
            const disabled = !asset.marketable;
            return (
              <li key={`${asset.type_id}-${asset.location_id}`}>
                <button
                  type="button"
                  data-testid="asset-row"
                  data-type-id={asset.type_id}
                  data-marketable={asset.marketable}
                  disabled={disabled}
                  onClick={() => handleSelect(asset)}
                  className={cn(
                    "w-full rounded-md border px-3 py-2 text-left text-sm transition-colors",
                    isSelected
                      ? "border-primary bg-primary/10"
                      : "border-transparent hover:bg-muted/60",
                    disabled && "cursor-not-allowed opacity-50"
                  )}
                  title={
                    disabled ? "Nicht handelbar (keine Marktgruppe)" : undefined
                  }
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium">{asset.name}</span>
                    <span className="text-muted-foreground">
                      {asset.quantity.toLocaleString("de-DE")}
                    </span>
                  </div>
                  <div className="truncate text-xs text-muted-foreground">
                    {asset.location_name}
                    {disabled && " · nicht handelbar"}
                  </div>
                </button>
              </li>
            );
          })}
        </ul>
      )}

      {selected && (
        <div className="space-y-4 border-t pt-4" data-testid="asset-sell-form">
          <div className="space-y-2">
            <Label htmlFor="sell-quantity">Menge</Label>
            <Input
              id="sell-quantity"
              type="number"
              min={1}
              max={selected.quantity}
              value={quantity}
              disabled={searching}
              onChange={(e) => setQuantity(Number(e.target.value))}
            />
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="avoid-low-sec"
              checked={avoidLowSec}
              disabled={searching}
              onCheckedChange={(c) => setAvoidLowSec(c === true)}
            />
            <Label htmlFor="avoid-low-sec" className="font-normal">
              Low-Sec meiden
            </Label>
          </div>

          <Button
            type="button"
            className="w-full"
            onClick={handleSubmit}
            disabled={searching || quantity < 1}
          >
            {searching && <Loader2 className="mr-2 size-4 animate-spin" />}
            {searching ? "Suche Verkaufsorte..." : "Verkaufsorte suchen"}
          </Button>
        </div>
      )}
    </div>
  );
}

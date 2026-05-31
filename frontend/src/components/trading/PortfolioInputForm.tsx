"use client";

import { RegionSelect } from "./RegionSelect";
import { CurrentShipCard } from "@/components/trading/CurrentShipCard";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Loader2 } from "lucide-react";

export interface PortfolioFormState {
  region: string;
  capital: number;
  timeBudgetMin: number;
  liquidityCapPct: number;
  maxItemPct: number;
  allowHighSec: boolean;
  allowLowSec: boolean;
  allowNullSec: boolean;
}

interface PortfolioInputFormProps {
  state: PortfolioFormState;
  onChange: (state: PortfolioFormState) => void;
  onSubmit: () => void;
  disabled?: boolean;
  loading?: boolean;
  /** Disables only the submit button (inputs stay editable). Used to block
   *  submission until the current ship is loaded. */
  submitDisabled?: boolean;
}

/**
 * Input form for the ROI Calculator / capital-allocation optimizer: region
 * selector, the read-only current-ship card, capital / time / liquidity /
 * max-per-item numeric inputs and security-zone checkboxes. The ship is no
 * longer user-selectable — it's always the character's current ship.
 */
export function PortfolioInputForm({
  state,
  onChange,
  onSubmit,
  disabled,
  loading,
  submitDisabled,
}: PortfolioInputFormProps) {
  const update = <K extends keyof PortfolioFormState>(
    key: K,
    value: PortfolioFormState[K]
  ) => onChange({ ...state, [key]: value });

  const noSecZone =
    !state.allowHighSec && !state.allowLowSec && !state.allowNullSec;

  return (
    <form
      className="space-y-4 rounded-lg border p-4"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
    >
      <RegionSelect
        value={state.region}
        onChange={(v) => update("region", v)}
        disabled={disabled}
      />
      <CurrentShipCard />

      <div className="space-y-2">
        <Label htmlFor="capital">Kapital (ISK)</Label>
        <Input
          id="capital"
          type="number"
          min={0}
          value={state.capital}
          disabled={disabled}
          onChange={(e) => update("capital", Number(e.target.value))}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="time-budget">Zeit-Budget (Min/Tag)</Label>
        <Input
          id="time-budget"
          type="number"
          min={0}
          value={state.timeBudgetMin}
          disabled={disabled}
          onChange={(e) => update("timeBudgetMin", Number(e.target.value))}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="liquidity-cap">Liquiditäts-Cap (%)</Label>
        <Input
          id="liquidity-cap"
          type="number"
          min={0}
          max={100}
          value={state.liquidityCapPct}
          disabled={disabled}
          onChange={(e) => update("liquidityCapPct", Number(e.target.value))}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="max-item">Max % Kapital pro Item</Label>
        <Input
          id="max-item"
          type="number"
          min={0}
          max={100}
          value={state.maxItemPct}
          disabled={disabled}
          onChange={(e) => update("maxItemPct", Number(e.target.value))}
        />
      </div>

      <fieldset className="space-y-2">
        <legend className="text-sm font-medium">Sicherheitszonen</legend>
        <div className="flex items-center gap-2">
          <Checkbox
            id="sec-high"
            checked={state.allowHighSec}
            disabled={disabled}
            onCheckedChange={(c) => update("allowHighSec", c === true)}
          />
          <Label htmlFor="sec-high" className="font-normal">
            High-Sec
          </Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="sec-low"
            checked={state.allowLowSec}
            disabled={disabled}
            onCheckedChange={(c) => update("allowLowSec", c === true)}
          />
          <Label htmlFor="sec-low" className="font-normal">
            Low-Sec
          </Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="sec-null"
            checked={state.allowNullSec}
            disabled={disabled}
            onCheckedChange={(c) => update("allowNullSec", c === true)}
          />
          <Label htmlFor="sec-null" className="font-normal">
            Null-Sec
          </Label>
        </div>
      </fieldset>

      <Button
        type="submit"
        className="w-full"
        disabled={disabled || loading || noSecZone || submitDisabled}
      >
        {loading && <Loader2 className="mr-2 size-4 animate-spin" />}
        {loading ? "Optimiere..." : "Portfolio optimieren"}
      </Button>
    </form>
  );
}

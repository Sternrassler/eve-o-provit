import { PortfolioFormState } from "@/components/trading/PortfolioInputForm";
import { PortfolioRequest, Ship } from "@/types/trading";

/**
 * Builds the optimizer request from the form state and the current ship.
 *
 * The ship is no longer user-selectable — it's the character's current ship,
 * mapped to the {@link Ship} shape. `ship_type_id` comes from that ship's
 * `type_id`. When no ship is available yet (e.g. still loading, or
 * unauthenticated), `ship_type_id` is `0`; callers should disable submission
 * until a ship is present so a `0` request is never sent.
 *
 * When the current ship's effective cargo is known (> 0 and not flagged
 * unavailable), that value is passed as `cargo_capacity` so the optimizer uses
 * the exact instance figure instead of a per-type recompute. Otherwise
 * `cargo_capacity` is omitted and the backend falls back.
 */
export function buildPortfolioRequest(
  form: PortfolioFormState,
  selectedShip: Ship | null,
): PortfolioRequest {
  const secZones: string[] = [];
  if (form.allowHighSec) secZones.push("high");
  if (form.allowLowSec) secZones.push("low");
  if (form.allowNullSec) secZones.push("null");

  const cargoCapacity =
    selectedShip &&
    !selectedShip.effective_cargo_unavailable &&
    selectedShip.effective_cargo_capacity != null &&
    selectedShip.effective_cargo_capacity > 0
      ? selectedShip.effective_cargo_capacity
      : undefined;

  return {
    region_id: parseInt(form.region, 10),
    ship_type_id: selectedShip ? selectedShip.type_id : 0,
    capital: form.capital,
    time_budget_min: form.timeBudgetMin,
    liquidity_cap_pct: form.liquidityCapPct,
    max_item_pct: form.maxItemPct,
    sec_zones: secZones,
    cargo_capacity: cargoCapacity,
  };
}

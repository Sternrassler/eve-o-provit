import { PortfolioFormState } from "@/components/trading/PortfolioInputForm";
import { PortfolioRequest, Ship } from "@/types/trading";

/**
 * Builds the optimizer request from the form state and the currently selected
 * ship instance.
 *
 * The form's `ship` value is now a ship instance's `item_id` (string), not a
 * type_id — so the type_id must come from `selectedShip.type_id`. When no ship
 * is selected (e.g. the unauthenticated default), the fallback ships have
 * `item_id === type_id`, so `parseInt(form.ship, 10)` still yields a valid
 * type_id.
 *
 * When a ship instance is selected and its effective cargo is known (> 0 and
 * not flagged unavailable), that value is passed as `cargo_capacity` so the
 * optimizer uses the exact instance figure instead of a per-type recompute.
 * Otherwise `cargo_capacity` is omitted and the backend falls back.
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
    ship_type_id: selectedShip ? selectedShip.type_id : parseInt(form.ship, 10),
    capital: form.capital,
    time_budget_min: form.timeBudgetMin,
    liquidity_cap_pct: form.liquidityCapPct,
    max_item_pct: form.maxItemPct,
    sec_zones: secZones,
    cargo_capacity: cargoCapacity,
  };
}

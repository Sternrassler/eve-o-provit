import { describe, it, expect } from "vitest";
import { buildPortfolioRequest } from "@/lib/portfolio-request";
import { PortfolioFormState } from "@/components/trading/PortfolioInputForm";
import { Ship } from "@/types/trading";

const baseForm: PortfolioFormState = {
  region: "10000002",
  ship: "111", // item_id of the selected instance (NOT a type_id)
  capital: 500_000_000,
  timeBudgetMin: 120,
  liquidityCapPct: 10,
  maxItemPct: 30,
  allowHighSec: true,
  allowLowSec: false,
  allowNullSec: false,
};

describe("buildPortfolioRequest", () => {
  it("sends the selected instance's effective cargo as cargo_capacity and its type_id", () => {
    const ship: Ship = {
      item_id: 111,
      type_id: 657,
      name: "Ix-ITE-1",
      cargo_capacity: 5800,
      effective_cargo_capacity: 6090,
    };

    const req = buildPortfolioRequest(baseForm, ship);

    expect(req.cargo_capacity).toBe(6090);
    // type_id must come from the selected ship, not from parseInt(form.ship)
    expect(req.ship_type_id).toBe(657);
  });

  it("omits cargo_capacity when the instance's effective cargo is unavailable", () => {
    const ship: Ship = {
      item_id: 111,
      type_id: 657,
      name: "Ix-ITE-1",
      cargo_capacity: 5800,
      effective_cargo_unavailable: true,
    };

    const req = buildPortfolioRequest(baseForm, ship);

    expect(req.cargo_capacity).toBeUndefined();
    expect(req.ship_type_id).toBe(657);
  });

  it("omits cargo_capacity and derives ship_type_id from form.ship when no ship is selected", () => {
    const form: PortfolioFormState = { ...baseForm, ship: "649" };

    const req = buildPortfolioRequest(form, null);

    expect(req.cargo_capacity).toBeUndefined();
    // Fallback ships have item_id === type_id, so parseInt is valid here.
    expect(req.ship_type_id).toBe(649);
  });

  it("maps security-zone flags to sec_zones", () => {
    const form: PortfolioFormState = {
      ...baseForm,
      allowHighSec: true,
      allowLowSec: true,
      allowNullSec: false,
    };

    const req = buildPortfolioRequest(form, null);

    expect(req.sec_zones).toEqual(["high", "low"]);
  });
});

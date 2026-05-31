import { describe, it, expect } from "vitest";
import { buildPortfolioRequest } from "@/lib/portfolio-request";
import { PortfolioFormState } from "@/components/trading/PortfolioInputForm";
import { Ship } from "@/types/trading";
import { CharacterShip } from "@/types/character";

const baseForm: PortfolioFormState = {
  region: "10000002",
  capital: 500_000_000,
  timeBudgetMin: 120,
  liquidityCapPct: 10,
  maxItemPct: 30,
  allowHighSec: true,
  allowLowSec: false,
  allowNullSec: false,
};

describe("buildPortfolioRequest", () => {
  it("sends the current ship's effective cargo as cargo_capacity and its type_id", () => {
    const ship: Ship = {
      item_id: 111,
      type_id: 657,
      name: "Ix-ITE-1",
      cargo_capacity: 5800,
      effective_cargo_capacity: 6090,
    };

    const req = buildPortfolioRequest(baseForm, ship);

    expect(req.cargo_capacity).toBe(6090);
    expect(req.ship_type_id).toBe(657);
  });

  it("omits cargo_capacity when the ship's effective cargo is unavailable", () => {
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

  it("sends ship_type_id: 0 and omits cargo_capacity when no ship is available", () => {
    const req = buildPortfolioRequest(baseForm, null);

    expect(req.cargo_capacity).toBeUndefined();
    // No ship loaded yet — callers must keep submission disabled in this state.
    expect(req.ship_type_id).toBe(0);
  });

  it("maps the current ship's effective cargo when mapped from CharacterShip", () => {
    // Mirrors the ROI page's mapping of useCurrentShip()'s CharacterShip into
    // the Ship shape buildPortfolioRequest expects.
    const currentShip: CharacterShip = {
      ship_type_id: 657,
      ship_name: "Ix-ITE-1",
      ship_item_id: 111,
      ship_type_name: "Iteron Mark V",
      cargo_capacity: 5800,
      effective_cargo_capacity: 6090,
    };
    const shipForRequest: Ship = {
      item_id: currentShip.ship_item_id,
      type_id: currentShip.ship_type_id,
      name: currentShip.ship_name,
      cargo_capacity: currentShip.cargo_capacity,
      effective_cargo_capacity: currentShip.effective_cargo_capacity,
      effective_cargo_unavailable: currentShip.effective_cargo_unavailable,
    };

    const req = buildPortfolioRequest(baseForm, shipForRequest);

    expect(req.ship_type_id).toBe(657);
    expect(req.cargo_capacity).toBe(6090);
  });

  it("maps security-zone flags to sec_zones", () => {
    const ship: Ship = {
      item_id: 111,
      type_id: 657,
      name: "Ix-ITE-1",
      cargo_capacity: 5800,
    };
    const form: PortfolioFormState = {
      ...baseForm,
      allowHighSec: true,
      allowLowSec: true,
      allowNullSec: false,
    };

    const req = buildPortfolioRequest(form, ship);

    expect(req.sec_zones).toEqual(["high", "low"]);
  });
});

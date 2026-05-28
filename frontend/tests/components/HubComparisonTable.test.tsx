import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { HubComparisonTable } from "@/components/trading/HubComparisonTable";
import { HubComparisonResult, HubRow } from "@/types/trading";

function makeHub(overrides: Partial<HubRow>): HubRow {
  return {
    region_id: 10000002,
    region_name: "The Forge",
    hub_name: "Jita",
    system_id: 30000142,
    tier: "primary",
    has_data: true,
    buy_price: 100,
    sell_price: 110,
    spread_percent: 10,
    net_margin_percent: 3.4,
    net_profit_per_unit: 4.1,
    daily_volume: 125000,
    liquidity_score: 72,
    competition: { changes_per_hour: 42.5, source: "baseline" },
    ...overrides,
  };
}

const result: HubComparisonResult = {
  type_id: 34,
  item_name: "Tritanium",
  best_hub_region_id: 10000032,
  skills_applied: {
    applied: true,
    accounting: 5,
    broker_relations: 5,
    sales_tax_rate: 0.025,
    broker_fee_rate: 0.015,
  },
  hubs: [
    makeHub({
      region_id: 10000032,
      region_name: "Sinq Laison",
      hub_name: "Dodixie",
      net_margin_percent: 12.0,
    }),
    makeHub({
      region_id: 10000002,
      region_name: "The Forge",
      hub_name: "Jita",
      net_margin_percent: 3.4,
    }),
    makeHub({
      region_id: 10000030,
      region_name: "Heimatar",
      hub_name: "Rens",
      has_data: false,
    }),
  ],
};

describe("HubComparisonTable", () => {
  it("renders a row for every hub including the item name", () => {
    render(<HubComparisonTable result={result} />);

    expect(screen.getByText(/Tritanium/)).toBeInTheDocument();
    expect(screen.getByText("Dodixie")).toBeInTheDocument();
    expect(screen.getByText("Jita")).toBeInTheDocument();
    expect(screen.getByText("Rens")).toBeInTheDocument();
    expect(screen.getAllByTestId("hub-row")).toHaveLength(3);
  });

  it("marks the recommended/best hub with the badge", () => {
    render(<HubComparisonTable result={result} />);

    const rows = screen.getAllByTestId("hub-row");
    const bestRow = rows.find(
      (r) => r.getAttribute("data-region-id") === "10000032"
    );
    const jitaRow = rows.find(
      (r) => r.getAttribute("data-region-id") === "10000002"
    );

    expect(bestRow).toBeDefined();
    expect(within(bestRow!).getByText("Empfohlen")).toBeInTheDocument();
    expect(within(jitaRow!).queryByText("Empfohlen")).not.toBeInTheDocument();
  });

  it("renders a no-data placeholder for hubs without data", () => {
    render(<HubComparisonTable result={result} />);

    const rows = screen.getAllByTestId("hub-row");
    const noDataRow = rows.find(
      (r) => r.getAttribute("data-region-id") === "10000030"
    );

    expect(noDataRow).toBeDefined();
    expect(
      within(noDataRow!).getByText("Keine Daten verfügbar")
    ).toBeInTheDocument();
  });
});

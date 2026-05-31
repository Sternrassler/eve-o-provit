import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { OreRankingTable } from "@/components/trading/OreRankingTable";
import { OreRankRow } from "@/types/trading";

function makeRow(overrides: Partial<OreRankRow> = {}): OreRankRow {
  return {
    ore_type_id: 1230,
    ore_name: "Veldspar",
    mining_m3_per_hour: 12000,
    raw_isk_per_hour: 500000,
    refine_isk_per_hour: 1100000,
    raw_net_per_m3: 76.8,
    refine_net_per_m3: 169.3,
    best: "refine",
    delta_isk_per_hour: 600000,
    best_station_id: 60003760,
    best_station_tax: 0.02,
    ...overrides,
  };
}

const rows: OreRankRow[] = [
  makeRow({ ore_type_id: 1230, ore_name: "Veldspar", best: "refine" }),
  makeRow({
    ore_type_id: 1228,
    ore_name: "Scordite",
    best: "raw",
    mining_m3_per_hour: 8000,
    raw_isk_per_hour: 900000,
    refine_isk_per_hour: 700000,
    delta_isk_per_hour: 200000,
    best_station_tax: 0.05,
  }),
];

describe("OreRankingTable", () => {
  it("renders one row per ore with correct ore names", () => {
    render(<OreRankingTable rows={rows} />);

    const tableRows = screen.getAllByTestId("ore-ranking-row");
    expect(tableRows).toHaveLength(2);
    expect(screen.getByText("Veldspar")).toBeInTheDocument();
    expect(screen.getByText("Scordite")).toBeInTheDocument();
  });

  it("renders 'Reprozessieren' verdict for best=refine row", () => {
    render(<OreRankingTable rows={rows} />);

    const veldsparRow = screen
      .getAllByTestId("ore-ranking-row")
      .find((r) => r.getAttribute("data-ore-type-id") === "1230")!;

    expect(within(veldsparRow).getByText("Reprozessieren")).toBeInTheDocument();
  });

  it("renders 'Roh verkaufen' verdict for best=raw row", () => {
    render(<OreRankingTable rows={rows} />);

    const scorditeRow = screen
      .getAllByTestId("ore-ranking-row")
      .find((r) => r.getAttribute("data-ore-type-id") === "1228")!;

    expect(within(scorditeRow).getByText("Roh verkaufen")).toBeInTheDocument();
  });

  it("renders tax percentage for each row", () => {
    render(<OreRankingTable rows={rows} />);

    expect(screen.getByText("2.0%")).toBeInTheDocument();
    expect(screen.getByText("5.0%")).toBeInTheDocument();
  });

  it("renders m³/h formatted with k suffix for values >= 1000", () => {
    render(<OreRankingTable rows={rows} />);

    // 12000 → "12.0k", 8000 → "8.0k"
    expect(screen.getByText("12.0k")).toBeInTheDocument();
    expect(screen.getByText("8.0k")).toBeInTheDocument();
  });

  it("renders the empty-state when rows array is empty", () => {
    render(<OreRankingTable rows={[]} />);

    expect(screen.getByTestId("ore-ranking-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("ore-ranking-row")).not.toBeInTheDocument();
  });
});

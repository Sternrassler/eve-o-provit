import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within, fireEvent } from "@testing-library/react";
import { OreRankingTable } from "@/components/trading/OreRankingTable";
import { OreRankRow } from "@/types/trading";
import { openMarketDetails, setWaypoint } from "@/lib/api-client";

vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({ toast: vi.fn() }),
}));
vi.mock("@/lib/api-client", () => ({
  openMarketDetails: vi.fn().mockResolvedValue(undefined),
  setWaypoint: vi.fn().mockResolvedValue(undefined),
}));

const openMarketMock = vi.mocked(openMarketDetails);
const setWaypointMock = vi.mocked(setWaypoint);

beforeEach(() => {
  openMarketMock.mockClear();
  setWaypointMock.mockClear();
});

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

  it("hides the detail panel until a row is clicked", () => {
    render(<OreRankingTable rows={rows} />);

    expect(screen.queryByTestId("ore-ranking-detail")).not.toBeInTheDocument();
  });

  it("expands a refine row to show the reprocess station and per-mineral sell breakdown", () => {
    const refineRows: OreRankRow[] = [
      makeRow({
        ore_type_id: 1230,
        ore_name: "Veldspar",
        best: "refine",
        best_station_name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
        best_station_system: "Jita",
        materials: [
          {
            material_type_id: 34,
            material_name: "Tritanium",
            effective_qty: 12345,
            buy_price: 5.5,
            sell: {
              station_name: "Amarr VIII - Emperor Family Academy",
              system_name: "Amarr",
              is_structure: false,
            },
          },
        ],
      }),
    ];
    render(<OreRankingTable rows={refineRows} />);

    fireEvent.click(screen.getByTestId("ore-ranking-row"));

    const detail = screen.getByTestId("ore-ranking-detail");
    expect(
      within(detail).getByText(/Caldari Navy Assembly Plant — Jita/),
    ).toBeInTheDocument();
    expect(within(detail).getByText("Tritanium")).toBeInTheDocument();
    expect(within(detail).getByText("12.345")).toBeInTheDocument(); // de-DE grouping
    expect(
      within(detail).getByText(/Emperor Family Academy — Amarr/),
    ).toBeInTheDocument();
  });

  it("expands a raw row to show the raw ore sell location", () => {
    const rawRows: OreRankRow[] = [
      makeRow({
        ore_type_id: 1228,
        ore_name: "Scordite",
        best: "raw",
        raw_sell: {
          station_name: "Dodixie IX - Moon 20 - Federation Navy Assembly Plant",
          system_name: "Dodixie",
          is_structure: false,
        },
      }),
    ];
    render(<OreRankingTable rows={rawRows} />);

    fireEvent.click(screen.getByTestId("ore-ranking-row"));

    const detail = screen.getByTestId("ore-ranking-detail");
    expect(
      within(detail).getByText(/Federation Navy Assembly Plant — Dodixie/),
    ).toBeInTheDocument();
  });

  it("marks an estimate row and omits the marker on an exact row", () => {
    const mixed: OreRankRow[] = [
      makeRow({ ore_type_id: 1230, ore_name: "Veldspar", is_estimate: false, hull_yield_multiplier: 1.495 }),
      makeRow({ ore_type_id: 1228, ore_name: "Scordite", is_estimate: true, estimate_reason: "Kein Crystal für dieses Erz" }),
    ];
    render(<OreRankingTable rows={mixed} />);

    const est = screen
      .getAllByTestId("ore-ranking-row")
      .find((r) => r.getAttribute("data-ore-type-id") === "1228")!;
    expect(within(est).getByTestId("ore-estimate-badge")).toBeInTheDocument();

    const exact = screen
      .getAllByTestId("ore-ranking-row")
      .find((r) => r.getAttribute("data-ore-type-id") === "1230")!;
    expect(within(exact).queryByTestId("ore-estimate-badge")).not.toBeInTheDocument();
  });

  it("renders 'Player-Structure' for a sell location inside a citadel", () => {
    const structureRows: OreRankRow[] = [
      makeRow({
        ore_type_id: 1228,
        ore_name: "Scordite",
        best: "raw",
        raw_sell: { is_structure: true },
      }),
    ];
    render(<OreRankingTable rows={structureRows} />);

    fireEvent.click(screen.getByTestId("ore-ranking-row"));

    const detail = screen.getByTestId("ore-ranking-detail");
    expect(within(detail).getByText("Player-Structure")).toBeInTheDocument();
  });

  it("shows effective ISK/h and cycle detail when present", () => {
    const r = makeRow({
      ore_type_id: 1230, ore_name: "Veldspar", best: "refine",
      effective_isk_per_hour: 950000, raw_isk_per_hour: 500000,
      refine_isk_per_hour: 1100000, cycle_minutes: 12.5, route_jumps: 3,
      sell_system_name: "Amarr",
    });
    render(<OreRankingTable rows={[r]} />);
    expect(screen.getByTestId("ore-effective-isk")).toHaveTextContent(/ISK/);

    fireEvent.click(screen.getByTestId("ore-ranking-row"));
    const detail = screen.getByTestId("ore-ranking-detail");
    expect(within(detail).getByText(/Zyklus/)).toBeInTheDocument();
    expect(within(detail).getByText(/Amarr/)).toBeInTheDocument();
  });

  it("opens the ore market on ore-name click without expanding the row", () => {
    render(<OreRankingTable rows={[makeRow({ ore_type_id: 1230, ore_name: "Veldspar" })]} />);
    fireEvent.click(screen.getByRole("button", { name: "Veldspar" }));
    expect(openMarketMock).toHaveBeenCalledWith(1230);
    expect(screen.queryByTestId("ore-ranking-detail")).not.toBeInTheDocument();
  });

  it("routes to the reprocess station and opens/routes minerals in the refine detail", () => {
    const r = makeRow({
      ore_type_id: 1230, ore_name: "Veldspar", best: "refine",
      best_station_id: 60003760, best_station_name: "Jita IV-4", best_station_system: "Jita",
      materials: [{
        material_type_id: 34, material_name: "Tritanium", effective_qty: 100, buy_price: 5,
        sell: { station_name: "Amarr VIII", system_name: "Amarr", is_structure: false, location_id: 60008494 },
      }],
    });
    render(<OreRankingTable rows={[r]} />);
    fireEvent.click(screen.getByTestId("ore-ranking-row"));

    fireEvent.click(screen.getByRole("button", { name: /Jita IV-4 — Jita/ }));
    expect(setWaypointMock).toHaveBeenCalledWith(60003760, { clearOtherWaypoints: true });

    fireEvent.click(screen.getByRole("button", { name: "Tritanium" }));
    expect(openMarketMock).toHaveBeenCalledWith(34);

    fireEvent.click(screen.getByRole("button", { name: /Amarr VIII — Amarr/ }));
    expect(setWaypointMock).toHaveBeenCalledWith(60008494, { clearOtherWaypoints: true });
  });

  it("routes to the raw ore sell location in the raw detail", () => {
    const r = makeRow({
      ore_type_id: 1228, ore_name: "Scordite", best: "raw",
      raw_sell: { station_name: "Dodixie IX", system_name: "Dodixie", is_structure: false, location_id: 60011866 },
    });
    render(<OreRankingTable rows={[r]} />);
    fireEvent.click(screen.getByTestId("ore-ranking-row"));
    fireEvent.click(screen.getByRole("button", { name: /Dodixie IX — Dodixie/ }));
    expect(setWaypointMock).toHaveBeenCalledWith(60011866, { clearOtherWaypoints: true });
  });
});

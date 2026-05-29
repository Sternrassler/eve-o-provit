import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within, fireEvent, act } from "@testing-library/react";
import { PortfolioResultTable } from "@/components/trading/PortfolioResultTable";
import { PortfolioItem, PortfolioResult } from "@/types/trading";
import { setWaypoint } from "@/lib/api-client";

vi.mock("@/lib/auth-context", () => ({
  useAuth: () => ({
    isAuthenticated: true,
  }),
}));

vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({
    toast: vi.fn(),
  }),
}));

vi.mock("@/lib/api-client", () => ({
  setWaypoint: vi.fn().mockResolvedValue(undefined),
}));

const setWaypointMock = vi.mocked(setWaypoint);

beforeEach(() => {
  setWaypointMock.mockClear();
});

function makeItem(overrides: Partial<PortfolioItem>): PortfolioItem {
  return {
    type_id: 34,
    name: "Tritanium",
    capital_used: 1.5e8,
    units: 120000,
    trips_per_day: 4,
    daily_profit: 2.5e7,
    roi_percent: 16.7,
    buy_system_name: "Jita",
    buy_station_name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
    buy_station_id: 60003760,
    sell_system_name: "Amarr",
    sell_station_name: "Amarr VIII (Oris) - Emperor Family Academy",
    sell_station_id: 60008494,
    ...overrides,
  };
}

const result: PortfolioResult = {
  items: [
    makeItem({ type_id: 34, name: "Tritanium", roi_percent: 16.7 }),
    makeItem({
      type_id: 35,
      name: "Pyerite",
      capital_used: 3.3e8,
      units: 90000,
      trips_per_day: 2,
      daily_profit: 3.6e7,
      roi_percent: 10.9,
    }),
  ],
  total_capital_used: 4.8e8,
  total_daily_profit: 6.1e7,
  time_used_min: 118,
  diversification_score: 72,
  skills_applied: {
    applied: true,
    accounting: 5,
    broker_relations: 5,
    sales_tax_rate: 0.025,
    broker_fee_rate: 0.015,
  },
};

describe("PortfolioResultTable", () => {
  it("renders one allocation row per item with name and ROI%", () => {
    render(<PortfolioResultTable result={result} />);

    const rows = screen.getAllByTestId("portfolio-row");
    expect(rows).toHaveLength(2);
    expect(screen.getByText("Tritanium")).toBeInTheDocument();
    expect(screen.getByText("Pyerite")).toBeInTheDocument();
    expect(screen.getByText("16.7%")).toBeInTheDocument();
    expect(screen.getByText("10.9%")).toBeInTheDocument();
  });

  it("renders units and trips per day for each item", () => {
    render(<PortfolioResultTable result={result} />);

    const tritRow = screen
      .getAllByTestId("portfolio-row")
      .find((r) => r.getAttribute("data-type-id") === "34");
    expect(tritRow).toBeDefined();
    expect(within(tritRow!).getByText("120.000")).toBeInTheDocument();
    expect(within(tritRow!).getByText("4")).toBeInTheDocument();
  });

  it("renders totals with full ISK precision", () => {
    render(<PortfolioResultTable result={result} />);

    const totals = screen.getByTestId("portfolio-totals");
    expect(within(totals).getByText("480.000.000,00 ISK")).toBeInTheDocument();
    expect(within(totals).getByText("61.000.000,00 ISK")).toBeInTheDocument();
  });

  it("renders the Route column as buy_system_name → sell_system_name with station tooltip", () => {
    render(<PortfolioResultTable result={result} />);

    const tritRow = screen
      .getAllByTestId("portfolio-row")
      .find((r) => r.getAttribute("data-type-id") === "34")!;

    const routeCell = within(tritRow).getByText(
      (_, el) => el?.textContent === "Jita → Amarr"
    );
    expect(routeCell).toBeInTheDocument();
    expect(routeCell).toHaveAttribute(
      "title",
      "Jita IV - Moon 4 - Caldari Navy Assembly Plant → Amarr VIII (Oris) - Emperor Family Academy"
    );
  });

  it("renders a waypoint button per row and calls setWaypoint with buy then sell station IDs on click", async () => {
    render(<PortfolioResultTable result={result} />);

    const buttons = screen.getAllByRole("button", {
      name: "Route an EVE übertragen",
    });
    expect(buttons).toHaveLength(2);

    const tritRow = screen
      .getAllByTestId("portfolio-row")
      .find((r) => r.getAttribute("data-type-id") === "34")!;
    const tritButton = within(tritRow).getByRole("button", {
      name: "Route an EVE übertragen",
    });

    await act(async () => {
      fireEvent.click(tritButton);
    });

    await vi.waitFor(() => {
      expect(setWaypointMock).toHaveBeenCalledTimes(2);
    });
    expect(setWaypointMock).toHaveBeenNthCalledWith(1, 60003760, {
      clearOtherWaypoints: true,
    });
    expect(setWaypointMock).toHaveBeenNthCalledWith(2, 60008494, {
      clearOtherWaypoints: false,
    });
  });

  it("renders the empty-state when no viable portfolio exists", () => {
    const empty: PortfolioResult = {
      ...result,
      items: [],
      total_capital_used: 0,
      total_daily_profit: 0,
    };
    render(<PortfolioResultTable result={empty} />);

    expect(screen.getByTestId("portfolio-empty")).toBeInTheDocument();
    expect(
      screen.getByText("Kein sinnvolles Portfolio für dieses Budget")
    ).toBeInTheDocument();
    expect(screen.queryByTestId("portfolio-row")).not.toBeInTheDocument();
  });
});

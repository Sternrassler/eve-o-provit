import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  render,
  screen,
  within,
  fireEvent,
  act,
} from "@testing-library/react";
import { HaulingRouteList } from "@/components/trading/HaulingRouteList";
import { HaulingRoute } from "@/types/trading";
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

function makeRoute(overrides: Partial<HaulingRoute> = {}): HaulingRoute {
  return {
    buy_system_name: "Osmon",
    buy_station_name: "Osmon V - Moon 5 - Sisters of EVE Bureau",
    buy_station_id: 60000001,
    sell_system_name: "Jita",
    sell_station_name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
    sell_station_id: 60003760,
    jumps: 7,
    travel_time_min: 12.5,
    security_risk: "safe",
    total_profit: 1.2e8,
    total_capital: 3.4e8,
    total_volume: 2400,
    isk_per_hour: 4.8e8,
    items: [
      {
        type_id: 34,
        name: "Meta-4 Module A",
        quantity: 50,
        buy_price: 1.0e6,
        sell_price: 1.9e6,
        unit_volume: 10,
        profit: 4.5e7,
        profit_per_m3: 90000,
      },
    ],
    ...overrides,
  };
}

describe("HaulingRouteList", () => {
  it("renders one card per route with the buy → sell systems, jumps, travel time and ISK/h", () => {
    render(<HaulingRouteList routes={[makeRoute()]} />);

    const cards = screen.getAllByTestId("hauling-route");
    expect(cards).toHaveLength(1);

    const card = cards[0];
    expect(
      within(card).getByText((_, el) => el?.textContent === "Osmon → Jita")
    ).toBeInTheDocument();
    expect(within(card).getByText("7 Sprünge")).toBeInTheDocument();
    expect(within(card).getByText("12.5 Min")).toBeInTheDocument();
    expect(
      within(card).getByText("480.000.000,00 ISK")
    ).toBeInTheDocument();
  });

  it("renders a security badge with the correct color per risk level", () => {
    const { rerender } = render(
      <HaulingRouteList routes={[makeRoute({ security_risk: "safe" })]} />
    );
    let badge = screen.getByTestId("security-badge");
    expect(badge).toHaveAttribute("data-risk", "safe");
    expect(badge.className).toContain("green");

    rerender(
      <HaulingRouteList routes={[makeRoute({ security_risk: "caution" })]} />
    );
    badge = screen.getByTestId("security-badge");
    expect(badge).toHaveAttribute("data-risk", "caution");
    expect(badge.className).toContain("amber");

    rerender(
      <HaulingRouteList routes={[makeRoute({ security_risk: "danger" })]} />
    );
    badge = screen.getByTestId("security-badge");
    expect(badge).toHaveAttribute("data-risk", "danger");
    expect(badge.className).toContain("red");
  });

  it("shows the cargo table only after expanding the route", () => {
    render(<HaulingRouteList routes={[makeRoute()]} />);

    expect(screen.queryByTestId("cargo-table")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("hauling-expand"));

    expect(screen.getByTestId("cargo-table")).toBeInTheDocument();
    const cargoRow = screen.getByTestId("cargo-row");
    expect(within(cargoRow).getByText("Meta-4 Module A")).toBeInTheDocument();
    // quantity 50, profit/m³ 90000
    expect(within(cargoRow).getByText("50")).toBeInTheDocument();
    expect(within(cargoRow).getByText("90.000,00 ISK")).toBeInTheDocument();
  });

  it("calls setWaypoint with buy then sell station IDs when pushing the route", async () => {
    render(<HaulingRouteList routes={[makeRoute()]} />);

    const button = screen.getByRole("button", {
      name: "Route an EVE übertragen",
    });

    await act(async () => {
      fireEvent.click(button);
    });

    await vi.waitFor(() => {
      expect(setWaypointMock).toHaveBeenCalledTimes(2);
    });
    expect(setWaypointMock).toHaveBeenNthCalledWith(1, 60000001, {
      clearOtherWaypoints: true,
    });
    expect(setWaypointMock).toHaveBeenNthCalledWith(2, 60003760, {
      clearOtherWaypoints: false,
    });
  });

  it("renders the empty-state when no routes exist", () => {
    render(<HaulingRouteList routes={[]} />);

    expect(screen.getByTestId("hauling-empty")).toBeInTheDocument();
    expect(
      screen.getByText("Keine profitablen Routen im Umkreis")
    ).toBeInTheDocument();
    expect(screen.queryByTestId("hauling-route")).not.toBeInTheDocument();
  });
});

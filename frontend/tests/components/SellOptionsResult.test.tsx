import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within, fireEvent, act } from "@testing-library/react";
import { SellOptionsResult } from "@/components/trading/SellOptionsResult";
import { SellOption } from "@/types/trading";
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

function makeOption(overrides: Partial<SellOption> = {}): SellOption {
  return {
    scope: "hub",
    region_id: 10000002,
    region_name: "The Forge",
    station_id: 60003760,
    station_name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
    system_name: "Jita",
    buy_price: 5.5,
    unit_net: 5.36,
    total_net: 643200,
    jumps: 0,
    travel_time_min: 0,
    security_risk: "safe",
    has_data: true,
    ...overrides,
  };
}

describe("SellOptionsResult", () => {
  it("renders an option card with system, station, scope and the money fields", () => {
    render(<SellOptionsResult best={null} options={[makeOption()]} />);

    const card = screen.getAllByTestId("sell-option")[0];
    expect(within(card).getByText("Jita")).toBeInTheDocument();
    expect(
      within(card).getByText(
        "Jita IV - Moon 4 - Caldari Navy Assembly Plant"
      )
    ).toBeInTheDocument();
    expect(within(card).getByTestId("scope-badge")).toHaveTextContent("Hub");
    // total_net 643200
    expect(within(card).getByText("643.200,00 ISK")).toBeInTheDocument();
    expect(within(card).getByText("0 Sprünge")).toBeInTheDocument();
  });

  it("renders a Region scope badge for current_region options", () => {
    render(
      <SellOptionsResult
        best={null}
        options={[makeOption({ scope: "current_region" })]}
      />
    );
    expect(screen.getByTestId("scope-badge")).toHaveTextContent("Region");
  });

  it("renders a security badge with the correct color per risk level", () => {
    const { rerender } = render(
      <SellOptionsResult
        best={null}
        options={[makeOption({ security_risk: "safe" })]}
      />
    );
    let badge = screen.getByTestId("security-badge");
    expect(badge).toHaveAttribute("data-risk", "safe");
    expect(badge.className).toContain("green");

    rerender(
      <SellOptionsResult
        best={null}
        options={[makeOption({ security_risk: "caution" })]}
      />
    );
    badge = screen.getByTestId("security-badge");
    expect(badge).toHaveAttribute("data-risk", "caution");
    expect(badge.className).toContain("amber");

    rerender(
      <SellOptionsResult
        best={null}
        options={[makeOption({ security_risk: "danger" })]}
      />
    );
    badge = screen.getByTestId("security-badge");
    expect(badge).toHaveAttribute("data-risk", "danger");
    expect(badge.className).toContain("red");
  });

  it("highlights the best option above the list", () => {
    const best = makeOption({ total_net: 700000 });
    render(<SellOptionsResult best={best} options={[makeOption()]} />);

    const all = screen.getAllByTestId("sell-option");
    // best card is rendered first, with the highlighted marker
    expect(all[0]).toHaveAttribute("data-highlighted", "true");
    expect(all[1]).not.toHaveAttribute("data-highlighted");
    expect(screen.getByLabelText("Beste Option")).toBeInTheDocument();
  });

  it("calls setWaypoint with the option's station_id and clearOtherWaypoints", async () => {
    render(<SellOptionsResult best={null} options={[makeOption()]} />);

    const button = screen.getByRole("button", {
      name: "Route an EVE übertragen",
    });

    await act(async () => {
      fireEvent.click(button);
    });

    await vi.waitFor(() => {
      expect(setWaypointMock).toHaveBeenCalledTimes(1);
    });
    expect(setWaypointMock).toHaveBeenCalledWith(60003760, {
      clearOtherWaypoints: true,
    });
  });

  it("renders has_data:false options muted with 'kein Marktpreis' and disabled waypoint", () => {
    render(
      <SellOptionsResult
        best={null}
        options={[makeOption({ has_data: false })]}
      />
    );

    const card = screen.getByTestId("sell-option");
    expect(card).toHaveAttribute("data-has-data", "false");
    expect(card.className).toContain("opacity-60");
    expect(screen.getByTestId("sell-option-no-data")).toHaveTextContent(
      "kein Marktpreis"
    );
    expect(
      screen.getByRole("button", { name: "Route an EVE übertragen" })
    ).toBeDisabled();
  });

  it("renders the empty-state when no options exist", () => {
    render(<SellOptionsResult best={null} options={[]} />);

    expect(screen.getByTestId("sell-options-empty")).toBeInTheDocument();
    expect(
      screen.getByText("Keine Verkaufsorte gefunden")
    ).toBeInTheDocument();
    expect(screen.queryByTestId("sell-option")).not.toBeInTheDocument();
  });
});

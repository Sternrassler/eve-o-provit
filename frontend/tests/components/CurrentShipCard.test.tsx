// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CurrentShipCard } from "@/components/trading/CurrentShipCard";

const mockRefresh = vi.fn();

vi.mock("@/lib/use-current-ship", () => ({
  useCurrentShip: vi.fn(),
}));

import { useCurrentShip } from "@/lib/use-current-ship";

const mockedUseCurrentShip = vi.mocked(useCurrentShip);

describe("CurrentShipCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the ship name and fitted cargo when effective cargo is present", () => {
    mockedUseCurrentShip.mockReturnValue({
      ship: {
        ship_type_id: 648,
        ship_item_id: 100000001,
        ship_name: "Mein Badger",
        ship_type_name: "Badger",
        cargo_capacity: 4000,
        effective_cargo_capacity: 6500,
        effective_cargo_unavailable: false,
      },
      isLoading: false,
      error: false,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    expect(screen.getByText(/Mein Badger/)).toBeInTheDocument();
    // 6500 m³ → "6.5k m³", no "Basis" prefix
    expect(screen.getByText(/6\.5k m³/)).toBeInTheDocument();
    expect(screen.queryByText(/Basis/)).not.toBeInTheDocument();
  });

  it("shows 'Basis … — fitted unbekannt' when effective_cargo_unavailable is true", () => {
    mockedUseCurrentShip.mockReturnValue({
      ship: {
        ship_type_id: 648,
        ship_item_id: 100000002,
        ship_name: "Sturdy Badger",
        ship_type_name: "Badger",
        cargo_capacity: 4000,
        effective_cargo_unavailable: true,
      },
      isLoading: false,
      error: false,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    expect(screen.getByText(/Sturdy Badger/)).toBeInTheDocument();
    expect(screen.getByText(/Basis 4\.0k m³ — fitted unbekannt/)).toBeInTheDocument();
  });

  it("shows 'Basis …' when no effective cargo is present", () => {
    mockedUseCurrentShip.mockReturnValue({
      ship: {
        ship_type_id: 648,
        ship_item_id: 100000003,
        ship_name: "Plain Badger",
        ship_type_name: "Badger",
        cargo_capacity: 4000,
      },
      isLoading: false,
      error: false,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    expect(screen.getByText(/Plain Badger/)).toBeInTheDocument();
    expect(screen.getByText(/Basis 4\.0k m³/)).toBeInTheDocument();
    expect(screen.queryByText(/fitted unbekannt/)).not.toBeInTheDocument();
  });

  it("calls refresh when the refresh button is clicked", async () => {
    const user = userEvent.setup();
    mockedUseCurrentShip.mockReturnValue({
      ship: {
        ship_type_id: 648,
        ship_item_id: 100000001,
        ship_name: "Mein Badger",
        ship_type_name: "Badger",
        cargo_capacity: 4000,
        effective_cargo_capacity: 6500,
      },
      isLoading: false,
      error: false,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    const btn = screen.getByTitle("Aktuelles Schiff neu laden");
    await user.click(btn);

    expect(mockRefresh).toHaveBeenCalledOnce();
  });

  it("shows loading placeholder when no ship and isLoading is true", () => {
    mockedUseCurrentShip.mockReturnValue({
      ship: null,
      isLoading: true,
      error: false,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    expect(screen.getByText("Lade aktuelles Schiff…")).toBeInTheDocument();
  });

  it("shows 'Kein aktives Schiff' when not authenticated / no ship", () => {
    mockedUseCurrentShip.mockReturnValue({
      ship: null,
      isLoading: false,
      error: false,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    expect(screen.getByText("Kein aktives Schiff")).toBeInTheDocument();
  });

  it("shows error alert (not 'Kein aktives Schiff') when fetch failed", () => {
    mockedUseCurrentShip.mockReturnValue({
      ship: null,
      isLoading: false,
      error: true,
      refresh: mockRefresh,
    });

    render(<CurrentShipCard />);

    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent(
      "Aktuelles Schiff konnte nicht geladen werden",
    );
    expect(screen.queryByText("Kein aktives Schiff")).not.toBeInTheDocument();
    // Refresh button must remain available for retry.
    expect(
      screen.getByTitle("Aktuelles Schiff neu laden"),
    ).toBeInTheDocument();
  });
});

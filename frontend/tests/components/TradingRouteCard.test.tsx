import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { TradingRouteCard } from "@/components/trading/TradingRouteCard";
import { TradingRoute } from "@/types/trading";

// Mock the auth context
vi.mock("@/lib/auth-context", () => ({
  useAuth: () => ({
    getAuthHeader: vi.fn(),
    isAuthenticated: false,
  }),
}));

// Mock the toast hook
vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({
    toast: vi.fn(),
  }),
}));

describe("TradingRouteCard - Fee Display", () => {
  const baseRoute: TradingRoute = {
    rank: 1,
    item_type_id: 34,
    item_name: "Tritanium",
    origin_system_name: "Jita",
    destination_system_name: "Amarr",
    quantity: 150000,
    buy_price: 5.50,
    sell_price: 7.85,
    total_investment: 825000,
    total_revenue: 1177500,
    profit: 352500,
    spread_percent: 42.7,
    travel_time_seconds: 0,
    round_trip_seconds: 0,
    isk_per_hour: 5250000,
  };

  it("displays fee breakdown when fee data is available", () => {
    const routeWithFees: TradingRoute = {
      ...baseRoute,
      gross_profit: 352500,
      gross_margin_percent: 42.7,
      sales_tax: 11775,
      broker_fees: 8837.5,
      estimated_relist_fee: 5887.5,
      total_fees: 26500,
      net_profit: 326000,
      net_profit_percent: 39.5,
    };

    render(<TradingRouteCard route={routeWithFees} />);

    // Check for fee section labels
    expect(screen.getByText("Brutto-Gewinn")).toBeInTheDocument();
    expect(screen.getByText("Brutto-Marge")).toBeInTheDocument();
    expect(screen.getByText("Gebühren")).toBeInTheDocument();
    expect(screen.getByText("Netto-Gewinn")).toBeInTheDocument();
    expect(screen.getByText("Netto-Marge")).toBeInTheDocument();
  });

  it("displays fallback profit display when fee data is not available", () => {
    render(<TradingRouteCard route={baseRoute} />);

    // Check for old display (without fee breakdown)
    expect(screen.getByText("Gewinn")).toBeInTheDocument();
    expect(screen.getByText("Spread")).toBeInTheDocument();
    
    // Should not show fee breakdown
    expect(screen.queryByText("Brutto-Gewinn")).not.toBeInTheDocument();
    expect(screen.queryByText("Gebühren")).not.toBeInTheDocument();
  });

  it("applies green color for high net margin (≥10%)", () => {
    const highMarginRoute: TradingRoute = {
      ...baseRoute,
      gross_profit: 352500,
      net_profit: 326000,
      net_profit_percent: 39.5,
      total_fees: 26500,
    };

    const { container } = render(<TradingRouteCard route={highMarginRoute} />);

    // Find elements with green color class
    const greenElements = container.querySelectorAll('.text-green-600');
    expect(greenElements.length).toBeGreaterThan(0);
  });

  it("applies yellow color for medium net margin (5-10%)", () => {
    const mediumMarginRoute: TradingRoute = {
      ...baseRoute,
      gross_profit: 59500,
      net_profit: 43592.5,
      net_profit_percent: 6.7,
      total_fees: 15907.5,
    };

    const { container } = render(<TradingRouteCard route={mediumMarginRoute} />);

    // Find elements with yellow color class
    const yellowElements = container.querySelectorAll('.text-yellow-600');
    expect(yellowElements.length).toBeGreaterThan(0);
  });

  it("applies red color for low net margin (<5%)", () => {
    const lowMarginRoute: TradingRoute = {
      ...baseRoute,
      gross_profit: 848250,
      net_profit: 558601.875,
      net_profit_percent: 4.6,
      total_fees: 289648.125,
    };

    const { container } = render(<TradingRouteCard route={lowMarginRoute} />);

    // Find elements with red color class
    const redElements = container.querySelectorAll('.text-red-600');
    expect(redElements.length).toBeGreaterThan(0);
  });

  it("formats ISK values with thousand separators", () => {
    const routeWithFees: TradingRoute = {
      ...baseRoute,
      gross_profit: 352500,
      net_profit: 326000,
      net_profit_percent: 39.5,
      total_fees: 26500,
    };

    render(<TradingRouteCard route={routeWithFees} />);

    // Check for formatted values (German locale: 352.500,00 ISK)
    expect(screen.getByText(/352\.500,00 ISK/)).toBeInTheDocument();
    expect(screen.getByText(/326\.000,00 ISK/)).toBeInTheDocument();
    expect(screen.getByText(/-26\.500,00 ISK/)).toBeInTheDocument();
  });
});

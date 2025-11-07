import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { FeeBreakdown } from "@/components/trading/FeeBreakdown";

describe("FeeBreakdown", () => {
  it("renders all fee components", () => {
    const fees = {
      salesTax: 11775,
      brokerFees: 8837.5,
      estimatedRelistFee: 5887.5,
      totalFees: 26500,
    };

    render(<FeeBreakdown fees={fees} />);

    // Check if labels are present
    expect(screen.getByText("Sales Tax:")).toBeInTheDocument();
    expect(screen.getByText("Broker Fees:")).toBeInTheDocument();
    expect(screen.getByText("Est. Relist Fee:")).toBeInTheDocument();
    expect(screen.getByText("Total:")).toBeInTheDocument();
  });

  it("formats ISK values with thousand separators", () => {
    const fees = {
      salesTax: 11775,
      brokerFees: 8837.5,
      estimatedRelistFee: 5887.5,
      totalFees: 26500,
    };

    render(<FeeBreakdown fees={fees} />);

    // Check if values are formatted correctly (German locale with thousand separators)
    expect(screen.getByText(/11\.775,00 ISK/)).toBeInTheDocument();
    expect(screen.getByText(/8\.837,50 ISK/)).toBeInTheDocument();
    expect(screen.getByText(/5\.887,50 ISK/)).toBeInTheDocument();
    expect(screen.getByText(/26\.500,00 ISK/)).toBeInTheDocument();
  });

  it("renders separator between fees and total", () => {
    const fees = {
      salesTax: 100,
      brokerFees: 200,
      estimatedRelistFee: 300,
      totalFees: 600,
    };

    const { container } = render(<FeeBreakdown fees={fees} />);

    // Check if separator is present
    const separator = container.querySelector('[data-slot="separator"]');
    expect(separator).toBeInTheDocument();
  });
});

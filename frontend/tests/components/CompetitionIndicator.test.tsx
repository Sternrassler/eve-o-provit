import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CompetitionIndicator } from "@/components/trading/CompetitionIndicator";
import { CompetitionInfo } from "@/types/trading";

describe("CompetitionIndicator", () => {
  it("shows the live label and rate for live-sourced data", () => {
    const competition: CompetitionInfo = {
      changes_per_hour: 42.5,
      source: "live",
    };

    render(<CompetitionIndicator competition={competition} />);

    expect(screen.getByText("Live")).toBeInTheDocument();
    expect(screen.getByText("42.5/h")).toBeInTheDocument();
    expect(screen.queryByText("Tages-Baseline")).not.toBeInTheDocument();
  });

  it("shows the baseline label for baseline-sourced data", () => {
    const competition: CompetitionInfo = {
      changes_per_hour: 3.2,
      source: "baseline",
    };

    render(<CompetitionIndicator competition={competition} />);

    expect(screen.getByText("Tages-Baseline")).toBeInTheDocument();
    expect(screen.getByText("3.2/h")).toBeInTheDocument();
    expect(screen.queryByText("Live")).not.toBeInTheDocument();
  });
});

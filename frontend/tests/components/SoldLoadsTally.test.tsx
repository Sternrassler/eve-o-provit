import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SoldLoadsTally } from "@/components/trading/SoldLoadsTally";

describe("SoldLoadsTally", () => {
  it("renders one pip per full load and the sold/total counter", () => {
    render(<SoldLoadsTally total={5} sold={2} onChange={() => {}} />);
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText(/\/ 5/)).toBeInTheDocument();
    // 5 pip buttons (group role).
    const group = screen.getByRole("group", { name: /verkaufte ladungen/i });
    expect(group.querySelectorAll("button")).toHaveLength(5);
  });

  it("tapping an unfilled pip fills up to it (sold = index + 1)", () => {
    const onChange = vi.fn();
    render(<SoldLoadsTally total={5} sold={2} onChange={onChange} />);
    // pip index 3 (4th) is unfilled (sold=2) → fills up to 4.
    fireEvent.click(screen.getByRole("button", { name: "Ladung 4" }));
    expect(onChange).toHaveBeenCalledWith(4);
  });

  it("tapping a filled pip clears from it (sold = index)", () => {
    const onChange = vi.fn();
    render(<SoldLoadsTally total={5} sold={3} onChange={onChange} />);
    // pip index 1 (2nd) is filled (sold=3) → clears down to 1.
    fireEvent.click(screen.getByRole("button", { name: /Ladung 2/ }));
    expect(onChange).toHaveBeenCalledWith(1);
  });

  it("switches to a +/- counter above the pip cap (24)", () => {
    const onChange = vi.fn();
    render(<SoldLoadsTally total={100} sold={42} onChange={onChange} />);
    expect(screen.queryByRole("group", { name: /verkaufte ladungen/i })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /eine ladung mehr/i }));
    expect(onChange).toHaveBeenCalledWith(43);
    fireEvent.click(screen.getByRole("button", { name: /eine ladung weniger/i }));
    expect(onChange).toHaveBeenCalledWith(41);
  });

  it("renders nothing when there are no full loads", () => {
    const { container } = render(<SoldLoadsTally total={0} sold={0} onChange={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });
});

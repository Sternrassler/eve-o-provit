// @vitest-environment jsdom
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { ShipSelect } from "@/components/trading/ShipSelect";

// Radix UI Select uses pointer-capture and scrollIntoView APIs that jsdom
// doesn't implement, causing crashes when the dropdown is opened. Mock the
// select primitives so we can test the data logic (dedupe by item_id) without
// fighting jsdom's limitations. Each SelectItem renders its children as a
// plain div, which lets us query by text content.
vi.mock("@/components/ui/select", () => ({
  Select: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SelectTrigger: ({ children }: { children: React.ReactNode }) => (
    <button>{children}</button>
  ),
  SelectValue: ({ placeholder }: { placeholder?: string }) => (
    <span>{placeholder}</span>
  ),
  SelectContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="select-content">{children}</div>
  ),
  SelectItem: ({
    children,
    value,
  }: {
    children: React.ReactNode;
    value: string;
  }) => (
    <div data-testid="select-item" data-value={value}>
      {children}
    </div>
  ),
}));

vi.mock("@/lib/api-client", () => ({
  fetchCharacterShips: vi.fn().mockResolvedValue([
    {
      item_id: 111,
      type_id: 657,
      name: "Ix-ITE-1",
      cargo_capacity: 5800,
      effective_cargo_capacity: 6090,
    },
    {
      item_id: 222,
      type_id: 657,
      name: "Ix-ITE-2",
      cargo_capacity: 5800,
      effective_cargo_capacity: 4900,
    },
  ]),
  fetchCharacterShip: vi.fn().mockResolvedValue({}),
}));

describe("ShipSelect", () => {
  it("renders one option per instance, not per type", async () => {
    render(<ShipSelect value="" onChange={() => {}} authenticated />);

    // Wait for ships to load and options to render
    await waitFor(() => {
      expect(screen.getAllByTestId("select-item").length).toBe(2);
    });

    // Both named instances of the same type_id must appear
    expect(screen.getByText(/Ix-ITE-1/)).toBeTruthy();
    expect(screen.getByText(/Ix-ITE-2/)).toBeTruthy();
  });

  it("uses item_id as option value (not type_id)", async () => {
    render(<ShipSelect value="" onChange={() => {}} authenticated />);

    await waitFor(() => {
      expect(screen.getAllByTestId("select-item").length).toBe(2);
    });

    const items = screen.getAllByTestId("select-item");
    const values = items.map((el) => el.getAttribute("data-value"));
    // item_ids are 111 and 222, NOT the shared type_id 657
    expect(values).toContain("111");
    expect(values).toContain("222");
    expect(values).not.toContain("657");
  });

  it("would collapse to one option if deduplicated by type_id (sanity check)", async () => {
    // This test documents WHY the fix matters: two ships with the same type_id
    // but different item_ids must both appear. If we deduped by type_id only
    // ONE would show — this test ensures both do.
    render(<ShipSelect value="" onChange={() => {}} authenticated />);

    await waitFor(() => {
      expect(screen.getAllByTestId("select-item").length).toBe(2);
    });

    // Two distinct items rendered — proves item_id dedupe (not type_id dedupe)
    const items = screen.getAllByTestId("select-item");
    expect(items).toHaveLength(2);
  });
});

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AssetPicker } from "@/components/trading/AssetPicker";
import { listAssets } from "@/lib/api-client";

vi.mock("@/lib/api-client", () => ({
  listAssets: vi.fn(),
}));

const listAssetsMock = vi.mocked(listAssets);

const assets = [
  { type_id: 1, name: "Tritanium", quantity: 5, location_id: 60000001, location_name: "A", system_id: 1, region_id: 1, marketable: true },
  { type_id: 2, name: "Pyerite", quantity: 100, location_id: 60000001, location_name: "A", system_id: 1, region_id: 1, marketable: true },
  { type_id: 3, name: "Mexallon", quantity: 50, location_id: 60000001, location_name: "A", system_id: 1, region_id: 1, marketable: true },
];

function renderPicker() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AssetPicker authenticated onSearch={vi.fn()} />
    </QueryClientProvider>
  );
}

function rowNames(): string[] {
  return within(screen.getByTestId("asset-list"))
    .getAllByTestId("asset-row")
    .map((el) => el.querySelector(".font-medium")?.textContent ?? "");
}

beforeEach(() => {
  listAssetsMock.mockReset();
  listAssetsMock.mockResolvedValue({ assets, count: assets.length });
});

describe("AssetPicker sorting", () => {
  it("sorts by name ascending by default", async () => {
    renderPicker();
    await waitFor(() => expect(screen.getByTestId("asset-list")).toBeInTheDocument());
    expect(rowNames()).toEqual(["Mexallon", "Pyerite", "Tritanium"]);
  });

  it("reverses order when toggling direction", async () => {
    renderPicker();
    await waitFor(() => expect(screen.getByTestId("asset-list")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("sort-dir"));
    expect(rowNames()).toEqual(["Tritanium", "Pyerite", "Mexallon"]);
  });

  it("sorts by quantity ascending", async () => {
    renderPicker();
    await waitFor(() => expect(screen.getByTestId("asset-list")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("sort-quantity"));
    expect(rowNames()).toEqual(["Tritanium", "Mexallon", "Pyerite"]); // 5, 50, 100
  });

  it("sorts by quantity descending", async () => {
    renderPicker();
    await waitFor(() => expect(screen.getByTestId("asset-list")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("sort-quantity"));
    fireEvent.click(screen.getByTestId("sort-dir"));
    expect(rowNames()).toEqual(["Pyerite", "Mexallon", "Tritanium"]); // 100, 50, 5
  });
});

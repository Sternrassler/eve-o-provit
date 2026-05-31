// @vitest-environment jsdom
import { describe, it, expect, afterEach, vi } from "vitest";
import { fetchCharacterShips } from "@/lib/api-client";

afterEach(() => vi.restoreAllMocks());

describe("fetchCharacterShips instances", () => {
  it("carries item_id and name per instance", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ ships: [
        { item_id: 111, type_id: 657, type_name: "Iteron Mark V", name: "Ix-ITE-1", cargo_capacity: 5800, effective_cargo_capacity: 6090 },
        { item_id: 222, type_id: 657, type_name: "Iteron Mark V", name: "Ix-ITE-2", cargo_capacity: 5800, effective_cargo_capacity: 4900 },
      ], count: 2 }), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    const ships = await fetchCharacterShips();
    expect(ships.map(s => s.item_id)).toEqual([111, 222]);
    expect(ships[0].name).toBe("Ix-ITE-1");
    expect(ships[1].effective_cargo_capacity).toBe(4900);
  });

  it("falls back to type_name when name is absent", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ ships: [
        { item_id: 333, type_id: 650, type_name: "Nereus", cargo_capacity: 2700 },
      ], count: 1 }), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    const ships = await fetchCharacterShips();
    expect(ships[0].name).toBe("Nereus");
  });
});

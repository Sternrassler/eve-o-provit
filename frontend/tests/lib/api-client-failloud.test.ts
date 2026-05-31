// @vitest-environment jsdom
import { describe, it, expect, afterEach, vi } from "vitest";
import {
  fetchCharacterShips,
  fetchRegions,
  searchItems,
} from "@/lib/api-client";

function mockJson(body: unknown, status = 200) {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("api-client fail-loud behaviour", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("fetchCharacterShips", () => {
    it("carries effective_cargo_capacity and the enrichment-error flag through", async () => {
      mockJson({
        ships: [
          {
            type_id: 16242,
            type_name: "Catalyst",
            cargo_capacity: 450,
            effective_cargo_capacity: 712,
          },
          {
            type_id: 650,
            type_name: "Bestower",
            cargo_capacity: 19000,
            effective_cargo_unavailable: true,
          },
        ],
        count: 2,
      });

      const ships = await fetchCharacterShips();

      expect(ships[0].effective_cargo_capacity).toBe(712);
      expect(ships[0].effective_cargo_unavailable).toBeUndefined();
      expect(ships[1].effective_cargo_capacity).toBeUndefined();
      expect(ships[1].effective_cargo_unavailable).toBe(true);
    });

    it("throws on a malformed response instead of silently returning []", async () => {
      mockJson({ count: 0 }); // no `ships` array
      await expect(fetchCharacterShips()).rejects.toThrow(/ships/);
    });
  });

  describe("fetchRegions", () => {
    it("throws on a malformed response instead of silently returning []", async () => {
      mockJson({ count: 0 }); // no `regions` array
      await expect(fetchRegions()).rejects.toThrow(/regions/);
    });
  });

  describe("searchItems", () => {
    it("throws on a malformed response instead of silently returning []", async () => {
      mockJson({}); // no `items` array
      await expect(searchItems("trit")).rejects.toThrow(/items/);
    });
  });
});

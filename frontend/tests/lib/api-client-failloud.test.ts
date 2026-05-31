// @vitest-environment jsdom
import { describe, it, expect, afterEach, vi } from "vitest";
import {
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

// @vitest-environment jsdom
import { describe, it, expect, afterEach, vi } from "vitest";
import { fetchOreRanking } from "@/lib/api-client";
import { OreRankRow } from "@/types/trading";

function mockJson(body: unknown, status = 200) {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

const makeRow = (overrides: Partial<OreRankRow> = {}): OreRankRow => ({
  ore_type_id: 1230,
  ore_name: "Veldspar",
  mining_m3_per_hour: 12000,
  raw_isk_per_hour: 0,
  refine_isk_per_hour: 0,
  raw_net_per_m3: 76.8,
  refine_net_per_m3: 169.3,
  best: "refine",
  delta_isk_per_hour: 0,
  best_station_id: 60003760,
  best_station_tax: 0.02,
  ...overrides,
});

describe("fetchOreRanking", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns mapped rows on a valid response", async () => {
    const rows = [makeRow(), makeRow({ ore_type_id: 1231, ore_name: "Scordite", best: "raw" })];
    mockJson({
      region_id: 10000002,
      sec_band: "high",
      no_mining_setup: false,
      rows,
    });

    const result = await fetchOreRanking({ region_id: 0, sec_band: "high" });

    expect(result.region_id).toBe(10000002);
    expect(result.sec_band).toBe("high");
    expect(result.no_mining_setup).toBe(false);
    expect(result.rows).toHaveLength(2);
    expect(result.rows[0].ore_name).toBe("Veldspar");
    expect(result.rows[1].ore_name).toBe("Scordite");
  });

  it("throws when the response is not ok", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ error: "unauthorized" }), {
        status: 401,
        statusText: "Unauthorized",
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(
      fetchOreRanking({ region_id: 0, sec_band: "high" }),
    ).rejects.toThrow(/Failed to fetch ore ranking/);
  });

  it("throws on a malformed response (no rows array) instead of returning silently", async () => {
    mockJson({ region_id: 10000002, sec_band: "high", no_mining_setup: false });
    // no `rows` key

    await expect(
      fetchOreRanking({ region_id: 0, sec_band: "high" }),
    ).rejects.toThrow(/rows/);
  });

  it("uses POST to /api/v1/mining/ore-ranking with credentials include", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({ region_id: 10000002, sec_band: "high", no_mining_setup: false, rows: [] }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    await fetchOreRanking({ region_id: 0, sec_band: "high" });

    expect(fetchSpy).toHaveBeenCalledOnce();
    const [url, init] = fetchSpy.mock.calls[0];
    expect(url).toContain("/api/v1/mining/ore-ranking");
    expect(init?.method).toBe("POST");
    expect(init?.credentials).toBe("include");
  });
});

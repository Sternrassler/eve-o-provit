import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { AuthProvider, useAuth } from "@/lib/auth-context";

vi.mock("@/lib/eve-sso", () => ({
  buildAuthorizationUrl: vi.fn(async () => "https://login.eveonline.com/authorize?mock=1"),
}));
import { buildAuthorizationUrl } from "@/lib/eve-sso";

function mockFetchSession(authenticated: boolean) {
  const character = authenticated
    ? {
        character_id: 90000001,
        character_name: "Test Pilot",
        scopes: ["esi-location.read_location.v1"],
        owner_hash: "ownerhash",
        portrait_url: "https://images.evetech.net/characters/90000001/portrait?size=128",
      }
    : null;
  return vi.fn(async (url: string, _init?: RequestInit) => {
    if (url.includes("/auth/session")) {
      return { ok: true, json: async () => ({ authenticated, character }) } as Response;
    }
    if (url.includes("/auth/logout")) {
      return { ok: true, json: async () => ({}) } as Response;
    }
    return { ok: true, json: async () => ({}) } as Response;
  });
}

describe("useAuth Hook (cookie session)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    process.env.NEXT_PUBLIC_EVE_CLIENT_ID = "test-client";
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns the auth context within AuthProvider", async () => {
    vi.stubGlobal("fetch", mockFetchSession(false));
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    expect(result.current).toBeDefined();
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.character).toBeNull();
    await waitFor(() => expect(result.current.isLoading).toBe(false));
  });

  it("throws when used outside an AuthProvider", () => {
    expect(() => renderHook(() => useAuth())).toThrow(
      "useAuth must be used within an AuthProvider"
    );
  });

  it("checks the session on mount via GET /auth/session with credentials", async () => {
    const fetchMock = mockFetchSession(false);
    vi.stubGlobal("fetch", fetchMock);
    renderHook(() => useAuth(), { wrapper: AuthProvider });
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalled();
    });
    const call = fetchMock.mock.calls.find((c) => String(c[0]).includes("/auth/session"));
    expect(call, "expected a /auth/session fetch").toBeTruthy();
    expect(call![1]).toMatchObject({ credentials: "include" });
  });

  it("populates character and isAuthenticated for an authenticated session", async () => {
    vi.stubGlobal("fetch", mockFetchSession(true));
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });
    expect(result.current.character?.character_id).toBe(90000001);
    expect(result.current.character?.character_name).toBe("Test Pilot");
  });

  it("logout calls POST /auth/logout and clears the character", async () => {
    const fetchMock = mockFetchSession(true);
    vi.stubGlobal("fetch", fetchMock);
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await result.current.logout();

    const logoutCall = fetchMock.mock.calls.find((c) => String(c[0]).includes("/auth/logout"));
    expect(logoutCall, "expected a /auth/logout fetch").toBeTruthy();
    expect(logoutCall![1]).toMatchObject({ method: "POST", credentials: "include" });
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.character).toBeNull();
    });
  });

  it("login builds the authorization URL and redirects", async () => {
    vi.stubGlobal("fetch", mockFetchSession(false));
    const original = window.location;
    delete (window as unknown as { location?: unknown }).location;
    (window as unknown as { location: { href: string } }).location = { href: "" };

    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await result.current.login();

    expect(buildAuthorizationUrl).toHaveBeenCalledWith(
      "test-client",
      expect.any(String),
      expect.arrayContaining(["esi-location.read_location.v1"])
    );
    expect(window.location.href).toContain("login.eveonline.com");

    (window as unknown as { location: Location }).location = original;
  });
});

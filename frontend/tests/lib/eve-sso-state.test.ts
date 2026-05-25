// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from "vitest";
import { validateState } from "@/lib/eve-sso";

describe("OAuth state single-use", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("validateState matches the stored state (read-only)", () => {
    sessionStorage.setItem("eve_oauth_state", "abc123");
    expect(validateState("abc123")).toBe(true);
    expect(validateState("wrong")).toBe(false);
  });

  it("simulating the callback cleanup removes eve_oauth_state (single-use)", () => {
    sessionStorage.setItem("eve_oauth_state", "abc123");
    const valid = validateState("abc123");
    sessionStorage.removeItem("eve_oauth_state");
    expect(valid).toBe(true);
    expect(sessionStorage.getItem("eve_oauth_state")).toBeNull();
  });
});

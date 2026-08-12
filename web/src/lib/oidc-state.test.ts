import { afterEach, describe, expect, it, vi } from "vitest";
import { createOidcState } from "./oidc-state";

describe("createOidcState", () => {
  afterEach(() => {
    document.cookie = "oidc_state=; Path=/; Max-Age=0";
    vi.unstubAllGlobals();
  });

  it("stores a random hex state in the oidc_state cookie", () => {
    const values = new Uint8Array(32).fill(0xab);
    vi.stubGlobal("crypto", {
      getRandomValues: (buf: Uint8Array) => {
        buf.set(values);
        return buf;
      },
    });

    const state = createOidcState();

    expect(state).toBe("ab".repeat(32));
    expect(document.cookie).toContain(`oidc_state=${state}`);
  });
});

import { describe, expect, it } from "vitest";
import { isOidcOnly } from "./auth-status";
import type { AuthStatus } from "@/lib/api/types";

function status(partial: Partial<AuthStatus>): AuthStatus {
  return {
    LoginEnabled: false,
    RegistrationEnabled: false,
    OIDCEnabled: false,
    ...partial,
  };
}

describe("isOidcOnly", () => {
  it("is false when status is missing", () => {
    expect(isOidcOnly(undefined)).toBe(false);
  });

  it("is true when login is disabled and OIDC is enabled", () => {
    expect(isOidcOnly(status({ OIDCEnabled: true }))).toBe(true);
  });

  it("is false when local login is enabled", () => {
    expect(isOidcOnly(status({ LoginEnabled: true, OIDCEnabled: true }))).toBe(false);
  });

  it("is false when OIDC is disabled", () => {
    expect(isOidcOnly(status({ LoginEnabled: false, OIDCEnabled: false }))).toBe(false);
  });
});

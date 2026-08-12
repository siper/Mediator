import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api, authApi, setAuthFailureHandler } from "@/lib/api/client";

interface MockResponse {
  ok: boolean;
  status: number;
  statusText: string;
  json: () => Promise<unknown>;
  text: () => Promise<string>;
}

function mockRes(status: number, body?: unknown): MockResponse {
  const text = body !== undefined ? JSON.stringify(body) : "";
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 401 ? "Unauthorized" : "OK",
    json: async () => (text ? JSON.parse(text) : {}),
    text: async () => text,
  };
}

function pathOf(url: unknown): string {
  if (typeof url === "string") return url;
  const u = url as { url?: string };
  return u?.url ?? String(url);
}

const originalFetch = globalThis.fetch;

beforeEach(() => {
  setAuthFailureHandler(() => {});
});

afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("request 401 refresh interceptor", () => {
  it("refreshes via POST /auth/refresh and retries the original request", async () => {
    const calls: { path: string; init?: RequestInit }[] = [];
    let dataCalls = 0;
    globalThis.fetch = vi.fn(async (url: unknown, init?: RequestInit) => {
      const path = pathOf(url);
      calls.push({ path, init });
      if (path === "/api/media") {
        dataCalls++;
        if (dataCalls === 1) return mockRes(401, { error: "expired" });
        return mockRes(200, [{ Id: "1" }]);
      }
      if (path === "/auth/refresh") return mockRes(200);
      throw new Error("unexpected " + path);
    }) as unknown as typeof fetch;

    const data = await api.get<unknown[]>("/api/media");

    expect(data).toEqual([{ Id: "1" }]);
    const refreshCalls = calls.filter((c) => c.path === "/auth/refresh");
    expect(refreshCalls).toHaveLength(1);
    expect(refreshCalls[0].init?.method).toBe("POST");
  });

  it("invokes the auth failure handler and rejects when refresh fails", async () => {
    const handler = vi.fn();
    setAuthFailureHandler(handler);
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const path = pathOf(url);
      if (path === "/api/media") return mockRes(401, { error: "expired" });
      if (path === "/auth/refresh") return mockRes(401, { error: "refresh expired" });
      throw new Error("unexpected " + path);
    }) as unknown as typeof fetch;

    await expect(api.get("/api/media")).rejects.toThrow();
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("runs a single refresh for concurrent 401 responses", async () => {
    let refreshCalls = 0;
    let aTries = 0;
    let bTries = 0;
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const path = pathOf(url);
      if (path === "/api/a") {
        aTries++;
        if (aTries === 1) return mockRes(401, { error: "expired" });
        return mockRes(200, { a: 1 });
      }
      if (path === "/api/b") {
        bTries++;
        if (bTries === 1) return mockRes(401, { error: "expired" });
        return mockRes(200, { b: 2 });
      }
      if (path === "/auth/refresh") {
        refreshCalls++;
        return mockRes(200);
      }
      throw new Error("unexpected " + path);
    }) as unknown as typeof fetch;

    const [a, b] = await Promise.all([api.get("/api/a"), api.get("/api/b")]);

    expect(a).toEqual({ a: 1 });
    expect(b).toEqual({ b: 2 });
    expect(refreshCalls).toBe(1);
  });

  it("does not refresh a second time when the retried request still fails with 401", async () => {
    let refreshCalls = 0;
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const path = pathOf(url);
      if (path === "/api/media") return mockRes(401, { error: "still expired" });
      if (path === "/auth/refresh") {
        refreshCalls++;
        return mockRes(200);
      }
      throw new Error("unexpected " + path);
    }) as unknown as typeof fetch;

    await expect(api.get("/api/media")).rejects.toThrow();
    expect(refreshCalls).toBe(1);
  });

  it("does not attempt refresh for auth endpoints", async () => {
    const calls: string[] = [];
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const path = pathOf(url);
      calls.push(path);
      return mockRes(401, { error: "bad credentials" });
    }) as unknown as typeof fetch;

    await expect(authApi.login("e", "p")).rejects.toThrow();
    expect(calls).toEqual(["/auth/login"]);
    expect(calls).not.toContain("/auth/refresh");
  });

  it("refreshes when /auth/me returns 401", async () => {
    let meCalls = 0;
    globalThis.fetch = vi.fn(async (url: unknown) => {
      const path = pathOf(url);
      if (path === "/auth/me") {
        meCalls++;
        if (meCalls === 1) return mockRes(401, { error: "expired" });
        return mockRes(200, { Id: "1", Email: "a@t.c", Role: "user", Name: "A", CreatedAt: "2026-01-01" });
      }
      if (path === "/auth/refresh") return mockRes(200);
      throw new Error("unexpected " + path);
    }) as unknown as typeof fetch;

    const user = await authApi.me();
    expect(user.Email).toBe("a@t.c");
    expect(meCalls).toBe(2);
  });
});

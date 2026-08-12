import type { AuthResponse, AuthStatus, User } from "@/lib/api/types";

const BASE = "";

let authFailureHandler: () => void = () => {};

export function setAuthFailureHandler(fn: () => void) {
  authFailureHandler = fn;
}

let refreshPromise: Promise<boolean> | null = null;

async function doRefresh(): Promise<boolean> {
  try {
    const res = await fetch(BASE + "/auth/refresh", {
      method: "POST",
      credentials: "same-origin",
    });
    if (!res.ok) {
      authFailureHandler();
      return false;
    }
    return true;
  } catch {
    authFailureHandler();
    return false;
  } finally {
    refreshPromise = null;
  }
}

function refreshOnce(): Promise<boolean> {
  if (!refreshPromise) refreshPromise = doRefresh();
  return refreshPromise;
}

export function refreshSession(): Promise<boolean> {
  return refreshOnce();
}

async function request<T>(path: string, init?: RequestInit, _retry = false): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    ...init,
  });
  if (!res.ok) {
    const canRefresh = path === "/auth/me" || !path.startsWith("/auth/");
    if (res.status === 401 && !_retry && canRefresh) {
      if (await refreshOnce()) return request<T>(path, init, true);
    }
    let message = res.statusText;
    try {
      const body = await res.json();
      message = body.error ?? message;
    } catch {
      // ignore
    }
    throw new Error(message);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};

export const authApi = {
  register: (name: string, email: string, password: string) =>
    request<AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ Name: name, Email: email, Password: password }),
    }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ Email: email, Password: password }),
    }),
  logout: () => request("/auth/logout", { method: "POST" }),
  me: () => request<User>("/auth/me"),
  status: () => request<AuthStatus>("/auth/status"),
  oidcLogin: (state: string) =>
    request<{ RedirectURL: string }>(`/auth/oidc/login?state=${encodeURIComponent(state)}`),
};

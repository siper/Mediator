import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import { ThemeProvider, useTheme } from "@/components/theme-provider";

const originalMatchMedia = window.matchMedia;

function installMatchMedia(matches: boolean) {
  window.matchMedia = ((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia;
}

const wrapper = ({ children }: { children: ReactNode }) => (
  <ThemeProvider>{children}</ThemeProvider>
);

describe("theme provider", () => {
  beforeEach(() => {
    installMatchMedia(false);
    localStorage.clear();
  });

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
  });

  it("defaults to system when nothing is stored", () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    expect(result.current.theme).toBe("system");
  });

  it("setTheme updates and persists the choice", () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    act(() => result.current.setTheme("dark"));
    expect(result.current.theme).toBe("dark");

    const { result: second } = renderHook(() => useTheme(), { wrapper });
    expect(second.current.theme).toBe("dark");
  });

  it("keeps system preference in sync with the OS", () => {
    const { result: first } = renderHook(() => useTheme(), { wrapper });
    act(() => first.current.setTheme("system"));

    installMatchMedia(true);
    const { result: second } = renderHook(() => useTheme(), { wrapper });
    expect(second.current.theme).toBe("system");
  });
});

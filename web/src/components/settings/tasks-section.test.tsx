import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { I18nextTestProvider } from "@/i18n";
import { TaskRow } from "@/components/settings/tasks-section";
import { api } from "@/lib/api/client";
import type { Task } from "@/lib/api/types";

const baseTask: Task = {
  Name: "refresh-metadata",
  Interval: "24h",
  Enabled: true,
  LastRun: "2026-08-04T10:00:00Z",
  NextRun: "2026-08-05T10:00:00Z",
  LastStatus: "ok",
  LastError: "",
};

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
  useMutation: (config: any) => ({
    mutate: vi.fn((...args: any[]) => {
      if (config.mutationFn) config.mutationFn(...args);
      if (config.onSuccess) config.onSuccess(undefined, ...args);
    }),
    isPending: false,
  }),
}));

vi.mock("@/lib/api/client", () => ({
  api: {
    put: vi.fn(),
    post: vi.fn(),
    get: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

describe("TaskRow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders task name and interval", () => {
    render(<I18nextTestProvider><TaskRow task={baseTask} /></I18nextTestProvider>);
    expect(screen.getByText("Refresh metadata")).toBeInTheDocument();
    expect(screen.getByDisplayValue("24h")).toBeInTheDocument();
  });

  it("renders enabled status as checked switch", () => {
    render(<I18nextTestProvider><TaskRow task={baseTask} /></I18nextTestProvider>);
    expect(screen.getByRole("switch", { checked: true })).toBeInTheDocument();
  });

  it("renders disabled status as unchecked switch", () => {
    render(<I18nextTestProvider><TaskRow task={{ ...baseTask, Enabled: false }} /></I18nextTestProvider>);
    expect(screen.getByRole("switch", { checked: false })).toBeInTheDocument();
  });

  it("renders status badge", () => {
    render(<I18nextTestProvider><TaskRow task={baseTask} /></I18nextTestProvider>);
    expect(screen.getByText("ok")).toBeInTheDocument();
  });

  it("renders error badge when LastError is set", () => {
    render(<I18nextTestProvider><TaskRow task={{ ...baseTask, LastStatus: "error", LastError: "DB connection failed" }} /></I18nextTestProvider>);
    expect(screen.getAllByText("error")).toHaveLength(2);
  });

  it("toggles enabled state on click", () => {
    render(<I18nextTestProvider><TaskRow task={baseTask} /></I18nextTestProvider>);
    const toggle = screen.getByRole("switch");
    fireEvent.click(toggle);
    expect(api.put).toHaveBeenCalledWith(`/api/tasks/${baseTask.Name}`, {
      interval: "24h",
      enabled: false,
    });
  });

  it("formats LastRun time", () => {
    render(<I18nextTestProvider><TaskRow task={baseTask} /></I18nextTestProvider>);
    expect(screen.getByText(/8\/4\/2026/)).toBeInTheDocument();
  });

  it("shows '—' for null times", () => {
    render(<I18nextTestProvider><TaskRow task={{ ...baseTask, LastRun: null, NextRun: null }} /></I18nextTestProvider>);
    const cells = screen.getAllByText("—");
    expect(cells.length).toBeGreaterThanOrEqual(2);
  });
});

import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { I18nextTestProvider } from "@/i18n";
import { AddMediaDialog } from "@/components/add-media-dialog";
import type { SearchResult } from "@/lib/api/types";

const albumTitle =
  "This Is A Very Long Music Album Title That Would Normally Overflow The Dialog Boundaries And Break The Layout";

const { mockResult, mock, apiPost } = vi.hoisted(() => ({
  mockResult: {
    ProviderName: "musicbrainz",
    ExternalID: "test-uuid",
    Title:
      "This Is A Very Long Music Album Title That Would Normally Overflow The Dialog Boundaries And Break The Layout",
    Overview: "A test album",
    CoverURL: "https://cover.example.com/test.jpg",
    MediaType: 3,
    Year: null,
  } as SearchResult,
  mock: {
    user: null as { Role: string } | null,
    publicSettings: { requests_enabled: "true" } as { requests_enabled?: string },
    lastMutate: null as null | {
      fn: (...a: unknown[]) => unknown;
      onSuccess?: (...a: unknown[]) => unknown;
      args: unknown[];
    },
  },
  apiPost: vi.fn(),
}));

vi.mock("@/store/ui", () => ({
  useUIStore: (selector: (s: { addResult: SearchResult | null; setAddResult: (r: SearchResult | null) => void }) => unknown) =>
    selector({ addResult: mockResult, setAddResult: vi.fn() }),
}));

vi.mock("@/store/auth", () => ({
  useAuthStore: (selector?: (s: { user: { Role: string } | null }) => unknown) =>
    selector ? selector({ user: mock.user }) : { user: mock.user },
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
  useQuery: (options: { queryKey?: string[] }) => {
    const key = options?.queryKey;
    if (key?.[0] === "settings" && key?.[1] === "public") {
      return { data: mock.publicSettings, isLoading: false, isError: false, error: null };
    }
    if (key?.[0] === "libraries") {
      return { data: [{ Id: 1, Name: "Lib", Type: mockResult.MediaType }], isLoading: false, isError: false, error: null };
    }
    if (key?.[0] === "profiles") {
      return { data: [{ Id: 1, Name: "Prof", Type: mockResult.MediaType }], isLoading: false, isError: false, error: null };
    }
    return { data: [], isLoading: false, isError: false, error: null };
  },
  useMutation: (options: {
    mutationFn?: (...a: unknown[]) => unknown;
    onSuccess?: (...a: unknown[]) => unknown;
  }) => ({
    isPending: false,
    mutate: (...args: unknown[]) => {
      mock.lastMutate = {
        fn: options?.mutationFn ?? (() => undefined),
        onSuccess: options?.onSuccess,
        args,
      };
    },
  }),
}));

vi.mock("@/lib/api/client", () => ({
  api: { get: vi.fn(), post: apiPost, put: vi.fn(), del: vi.fn(), patch: vi.fn() },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

function resetResult(overrides: Partial<SearchResult> = {}) {
  mockResult.ProviderName = "musicbrainz";
  mockResult.ExternalID = "test-uuid";
  mockResult.Title = albumTitle;
  mockResult.Overview = "A test album";
  mockResult.CoverURL = "https://cover.example.com/test.jpg";
  mockResult.MediaType = 3;
  mockResult.Year = null;
  Object.assign(mockResult, overrides);
}

describe("AddMediaDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mock.user = null;
    mock.publicSettings = { requests_enabled: "true" };
    mock.lastMutate = null;
    resetResult();
  });

  it("renders with a long music album title", () => {
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByText(mockResult.Title)).toBeInTheDocument();
  });

  it("applies min-w-0 to title for proper truncation in flex column layout", () => {
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    const title = screen.getByText(mockResult.Title);
    expect(title).toHaveClass("min-w-0");
    expect(title).toHaveClass("truncate");
  });

  it("uses wider dialog (max-w-lg) instead of max-w-md", () => {
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    const dialog = screen.getByRole("dialog");
    expect(dialog).not.toHaveClass("max-w-md");
  });

  it("applies overflow-hidden to flex container to prevent grid item overflow", () => {
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    const title = screen.getByText(mockResult.Title);
    const outerFlex = title.closest(".flex.overflow-hidden");
    expect(outerFlex).toBeInTheDocument();
    expect(outerFlex).toHaveClass("overflow-hidden");
  });

  it("pre-fills folder with title and year", () => {
    resetResult({ Title: "The Matrix", Year: 1999 });
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByPlaceholderText("Folder name (optional, defaults to title)")).toHaveValue("The Matrix (1999)");
  });

  it("pre-fills folder with title when year is missing", () => {
    resetResult({ Title: "Test Book", Year: null });
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByPlaceholderText("Folder name (optional, defaults to title)")).toHaveValue("Test Book");
  });

  it("shows Import button and posts to /api/providers/import for admin", async () => {
    mock.user = { Role: "admin" };
    mock.publicSettings = { requests_enabled: "true" };
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Import" }));
    await waitFor(() => expect(mock.lastMutate).not.toBeNull());
    await mock.lastMutate!.fn(...mock.lastMutate!.args);
    expect(apiPost).toHaveBeenCalledWith(
      "/api/providers/import",
      expect.objectContaining({ provider: "musicbrainz", external_id: "test-uuid" }),
    );
  });

  it("renders singular mediaType label for movie (not mediaTypeNav slug)", () => {
    resetResult({ MediaType: 2, ProviderName: "tmdb", Title: "Test Movie" });
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByText("Movie")).toBeInTheDocument();
    expect(screen.queryByText("mediaType.movies")).not.toBeInTheDocument();
  });

  it("renders singular mediaType label for book", () => {
    resetResult({ MediaType: 0, ProviderName: "author_today", Title: "Test Book" });
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByText("Book")).toBeInTheDocument();
    expect(screen.queryByText("mediaType.books")).not.toBeInTheDocument();
  });

  it("shows Import button for non-admin when requests disabled", async () => {
    mock.user = { Role: "user" };
    mock.publicSettings = { requests_enabled: "false" };
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByRole("button", { name: "Import" })).toBeInTheDocument();
  });

  it("shows Request button and posts to /api/requests for non-admin when requests enabled", async () => {
    mock.user = { Role: "user" };
    mock.publicSettings = { requests_enabled: "true" };
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Request" }));
    await waitFor(() => expect(mock.lastMutate).not.toBeNull());
    await mock.lastMutate!.fn(...mock.lastMutate!.args);
    expect(apiPost).toHaveBeenCalledWith(
      "/api/requests",
      expect.objectContaining({
        provider: "musicbrainz",
        external_id: "test-uuid",
        type: mockResult.MediaType,
        library_id: 1,
        quality_profile_id: 1,
        folder: albumTitle,
      }),
    );
    expect(apiPost).toHaveBeenCalledTimes(1);
    expect(apiPost.mock.calls[0][0]).toBe("/api/requests");
  });

  it("hides search-missing toggle in request mode", () => {
    mock.user = { Role: "user" };
    mock.publicSettings = { requests_enabled: "true" };
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.queryByText("Search for missing album")).not.toBeInTheDocument();
  });

  it.each([
    { MediaType: 0 as const, ProviderName: "author_today", Title: "Test Book", label: "Search for missing book" },
    { MediaType: 1 as const, ProviderName: "tmdb", Title: "Test Series", label: "Search for missing episodes" },
    { MediaType: 2 as const, ProviderName: "tmdb", Title: "Test Movie", label: "Search for missing movie" },
    { MediaType: 3 as const, ProviderName: "musicbrainz", Title: "Test Album", label: "Search for missing album" },
  ])("shows search-missing toggle for $label when importing", ({ MediaType, ProviderName, Title, label }) => {
    mock.user = { Role: "admin" };
    mock.publicSettings = { requests_enabled: "true" };
    resetResult({ MediaType, ProviderName, Title });
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("posts grab-missing after import when search toggle is on", async () => {
    mock.user = { Role: "admin" };
    mock.publicSettings = { requests_enabled: "true" };
    resetResult({ MediaType: 2, ProviderName: "tmdb", Title: "Test Movie" });
    render(<I18nextTestProvider><AddMediaDialog /></I18nextTestProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Import" }));
    await waitFor(() => expect(mock.lastMutate).not.toBeNull());
    const importMut = mock.lastMutate!;
    await importMut.fn(...importMut.args);
    importMut.onSuccess?.({ Id: 42, Name: "Test Movie", Type: 2 });
    await waitFor(() => expect(mock.lastMutate).not.toBe(importMut));
    await mock.lastMutate!.fn(...mock.lastMutate!.args);
    expect(apiPost).toHaveBeenCalledWith("/api/media/42/grab-missing");
  });
});

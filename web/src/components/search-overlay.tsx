import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Search, X, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Library, Media, SearchPage, SearchResult } from "@/lib/api/types";
import { MediaType, MEDIA_TYPE_SLUG } from "@/lib/api/types";
import { CoverImg } from "@/components/ui/cover-img";
import { MediaTypeBadge } from "@/components/media-type-badge";
import { useUIStore } from "@/store/ui";

const TYPE_ORDER: MediaType[] = [
  MediaType.Book,
  MediaType.Series,
  MediaType.Movie,
  MediaType.MusicAlbum,
];

const PROVIDER_PAGE_SIZE = 25;

export function SearchOverlay() {
  const { t } = useTranslation();
  const open = useUIStore((s) => s.searchOpen);
  const setOpen = useUIStore((s) => s.setSearchOpen);
  const setAddResult = useUIStore((s) => s.setAddResult);
  const searchType = useUIStore((s) => s.searchType);
  const setSearchType = useUIStore((s) => s.setSearchType);
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");

  useEffect(() => {
    if (!open) return;
    setQuery("");
    setDebounced("");
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [open]);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 300);
    return () => clearTimeout(t);
  }, [query]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    if (open) window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, setOpen]);

  const localQuery = useQuery({
    queryKey: ["media", "all"],
    queryFn: () => api.get<Media[]>("/api/media?limit=100"),
    enabled: open,
  });

  const librariesQuery = useQuery({
    queryKey: ["libraries"],
    queryFn: () => api.get<Library[]>("/api/libraries"),
    enabled: open,
  });

  const availableTypes = useMemo(() => {
    if (!librariesQuery.data) return null;
    return new Set(librariesQuery.data.map((l) => l.Type));
  }, [librariesQuery.data]);

  const typeTabs = useMemo(() => {
    if (!availableTypes) return [];
    return TYPE_ORDER.filter((tt) => availableTypes.has(tt));
  }, [availableTypes]);

  const activeTypes = useMemo(() => {
    if (!availableTypes) return null;
    if (searchType !== null) return new Set<MediaType>([searchType]);
    return availableTypes;
  }, [availableTypes, searchType]);

  const showProviders = debounced.trim().length > 1;

  const providerQuery = useInfiniteQuery({
    queryKey: ["providers/search", debounced, searchType],
    queryFn: ({ pageParam }) => {
      const params = new URLSearchParams({
        q: debounced,
        page: String(pageParam),
        limit: String(PROVIDER_PAGE_SIZE),
      });
      if (searchType !== null) {
        params.set("type", String(searchType));
      }
      return api.get<SearchPage>(`/api/providers/search?${params}`);
    },
    initialPageParam: 1,
    getNextPageParam: (last) => (last.HasMore ? last.Page + 1 : undefined),
    enabled: open && showProviders,
  });

  const localResults = useMemo(() => {
    const q = debounced.trim().toLowerCase();
    const all = localQuery.data ?? [];
    const filtered =
      activeTypes === null ? all : all.filter((m) => activeTypes.has(m.Type));
    if (!q) return filtered.slice(0, 8);
    return filtered.filter((m) => m.Name.toLowerCase().includes(q)).slice(0, 8);
  }, [localQuery.data, debounced, activeTypes]);

  const providerResults = useMemo(() => {
    const pages = providerQuery.data?.pages ?? [];
    const raw = pages.flatMap((p) => p.Results ?? []);
    const seen = new Set<string>();
    const unique: SearchResult[] = [];
    for (const r of raw) {
      const key = `${r.ProviderName}:${r.ExternalID}`;
      if (seen.has(key)) continue;
      seen.add(key);
      if (activeTypes !== null && !activeTypes.has(r.MediaType)) continue;
      unique.push(r);
    }
    return unique;
  }, [providerQuery.data, activeTypes]);

  useEffect(() => {
    const root = listRef.current;
    const target = sentinelRef.current;
    if (!root || !target || !showProviders) return;

    const obs = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        if (providerQuery.hasNextPage && !providerQuery.isFetchingNextPage) {
          void providerQuery.fetchNextPage();
        }
      },
      { root, rootMargin: "120px", threshold: 0 }
    );
    obs.observe(target);
    return () => obs.disconnect();
  }, [
    showProviders,
    providerQuery.hasNextPage,
    providerQuery.isFetchingNextPage,
    providerQuery.fetchNextPage,
    providerResults.length,
  ]);

  const goMedia = (m: Media) => {
    setOpen(false);
    if (m.Type === MediaType.Movie) navigate(`/movies/${m.Id}`);
    else if (m.Type === MediaType.Series) navigate(`/series/${m.Id}`);
    else if (m.Type === MediaType.Book) navigate(`/books/${m.Id}`);
    else if (m.Type === MediaType.MusicAlbum) navigate(`/music/${m.Id}`);
    else navigate(`/${MEDIA_TYPE_SLUG[m.Type] ?? String(m.Type)}`);
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 px-4 pt-24">
      <div
        className="absolute inset-0"
        onClick={() => setOpen(false)}
        aria-hidden
      />
      <div className="relative z-10 w-full max-w-xl overflow-hidden rounded-lg border border-border bg-popover shadow-[0_10px_28px_rgb(0_0_0/0.18),0_1px_2px_rgb(0_0_0/0.01)]">
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("search.placeholder")}
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          <button
            onClick={() => setOpen(false)}
            aria-label={t("common.close")}
            className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent"
          >
            <X className="size-4" />
          </button>
        </div>

        {typeTabs.length > 1 && (
          <div className="-mx-1 border-b border-border px-3">
            <div className="no-scrollbar flex items-center gap-1 overflow-x-auto py-2">
              <Tab
                label={t("search.all")}
                active={searchType === null}
                onClick={() => setSearchType(null)}
              />
              {typeTabs.map((tt) => (
                <Tab
                  key={tt}
                  label={t(`mediaTypeNav.${MEDIA_TYPE_SLUG[tt]}`)}
                  active={searchType === tt}
                  onClick={() => setSearchType(tt)}
                />
              ))}
            </div>
          </div>
        )}

        <div ref={listRef} className="max-h-[60vh] overflow-y-auto p-2">
          {!showProviders && localResults.length === 0 && (
            <p className="px-3 py-8 text-center text-sm text-muted-foreground">
              {localQuery.isLoading ? t("empty.loadingLibrary") : t("empty.library")}
            </p>
          )}

          {localResults.length > 0 && (
            <Group title={t("search.libraryGroup")}>
              {localResults.map((m) => (
                <button
                  key={m.Id}
                  onClick={() => goMedia(m)}
                  className="flex w-full cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-accent"
                >
                  <div className={`w-10 shrink-0 overflow-hidden rounded-md bg-muted ${posterAspect(m.Type)}`}>
                    <CoverImg
                      src={m.Cover ?? ""}
                      alt={m.Name}
                      className="h-full w-full object-cover"
                    />
                  </div>
                  <span className="min-w-0 flex-1 truncate font-medium">{m.Name}</span>
                  <MediaTypeBadge type={m.Type} />
                </button>
              ))}
            </Group>
          )}

          {showProviders && (
            <Group title={t("search.providersGroup")}>
              {providerQuery.isLoading && (
                <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                  <Loader2 className="size-4 animate-spin" /> {t("search.searching")}
                </div>
              )}
              {providerQuery.isError && (
                <p className="px-3 py-3 text-sm text-destructive">
                  {(providerQuery.error as Error).message}
                </p>
              )}
              {!providerQuery.isLoading &&
                providerResults.length === 0 &&
                !providerQuery.isError && (
                  <p className="px-3 py-3 text-sm text-muted-foreground">
                    {t("search.noResults")}
                  </p>
                )}
              {providerResults.map((r) => (
                <button
                  key={`${r.ProviderName}:${r.ExternalID}`}
                  onClick={() => setAddResult(r)}
                  className="flex w-full cursor-pointer items-start gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-accent"
                >
                  <div className={`w-14 shrink-0 overflow-hidden rounded-md bg-muted ${posterAspect(r.MediaType)}`}>
                    <CoverImg
                      src={r.CoverURL}
                      alt={r.Title}
                      className="h-full w-full object-cover"
                    />
                  </div>
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <p className="line-clamp-1 font-medium">{r.Title}</p>
                    {r.Overview && (
                      <p className="line-clamp-2 text-xs text-muted-foreground">{r.Overview}</p>
                    )}
                    <span className="text-xs text-muted-foreground">{r.ProviderName}</span>
                  </div>
                  <MediaTypeBadge type={r.MediaType} className="shrink-0" />
                </button>
              ))}
              <div ref={sentinelRef} className="h-1" />
              {providerQuery.isFetchingNextPage && (
                <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                  <Loader2 className="size-4 animate-spin" /> {t("search.loadingMore")}
                </div>
              )}
            </Group>
          )}
        </div>
      </div>
    </div>
  );
}

function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="px-2 py-1">
      <p className="px-3 pb-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {title}
      </p>
      {children}
    </div>
  );
}

function Tab({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={
        active
          ? "rounded-md border border-border bg-accent px-3 py-1.5 text-sm font-medium text-foreground"
          : "rounded-md border border-transparent px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      }
    >
      {label}
    </button>
  );
}

function posterAspect(type: MediaType): string {
  if (type === MediaType.MusicAlbum) return "aspect-square";
  return type === MediaType.Movie || type === MediaType.Series
    ? "aspect-[2/3]"
    : "aspect-[3/4]";
}

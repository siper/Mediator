import { useEffect, useMemo, useRef } from "react";
import { Link } from "react-router-dom";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { HistoryListItem } from "@/lib/api/types";
import { MEDIA_TYPE_SLUG, MediaType } from "@/lib/api/types";
import { CoverImg } from "@/components/ui/cover-img";
import { Spinner } from "@/components/ui/spinner";
import { MediaTypeBadge } from "@/components/media-type-badge";
import { HistoryEventBadge } from "@/components/history-event-badge";

const PAGE_SIZE = 20;

function posterAspect(type: MediaType): string {
  if (type === MediaType.MusicAlbum) return "aspect-square";
  return type === MediaType.Movie || type === MediaType.Series
    ? "aspect-[2/3]"
    : "aspect-[3/4]";
}

export function HistoryPage() {
  const { t } = useTranslation();
  const sentinelRef = useRef<HTMLDivElement>(null);

  const historyQuery = useInfiniteQuery({
    queryKey: ["history", "infinite"],
    queryFn: ({ pageParam }) =>
      api.get<HistoryListItem[]>(
        `/api/history?page=${pageParam}&limit=${PAGE_SIZE}`
      ),
    initialPageParam: 1,
    getNextPageParam: (last, pages) =>
      last.length < PAGE_SIZE ? undefined : pages.length + 1,
  });

  const items = useMemo(
    () => (historyQuery.data?.pages ?? []).flat(),
    [historyQuery.data]
  );

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target) return;

    const obs = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        if (historyQuery.hasNextPage && !historyQuery.isFetchingNextPage) {
          void historyQuery.fetchNextPage();
        }
      },
      { root: null, rootMargin: "120px", threshold: 0 }
    );
    obs.observe(target);
    return () => obs.disconnect();
  }, [
    historyQuery.hasNextPage,
    historyQuery.isFetchingNextPage,
    historyQuery.fetchNextPage,
    items.length,
  ]);

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <header className="mb-8">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t("history.title")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("history.description")}
            </p>
          </div>
          {!historyQuery.isLoading && items.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {t("history.count", { count: items.length })}
            </p>
          )}
        </div>
      </header>

      {historyQuery.isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner /> {t("common.loading")}
        </div>
      )}
      {historyQuery.isError && (
        <p className="text-sm text-destructive">
          {(historyQuery.error as Error).message}
        </p>
      )}

      {!historyQuery.isLoading && items.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
          <p className="text-sm text-muted-foreground">{t("empty.noActivity")}</p>
        </div>
      )}

      {items.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-border bg-popover">
          <ul className="divide-y divide-border">
            {items.map((h) => (
              <HistoryRow key={h.Id} item={h} />
            ))}
          </ul>
          <div ref={sentinelRef} className="h-1" />
          {historyQuery.isFetchingNextPage && (
            <div className="flex items-center gap-2 px-4 py-3 text-sm text-muted-foreground">
              <Spinner /> {t("search.loadingMore")}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function HistoryRow({ item }: { item: HistoryListItem }) {
  const { t } = useTranslation();
  const media = item.Media;
  const hasMedia = media?.Id > 0 && !!media.Name;
  const type = media?.Type ?? MediaType.Movie;
  const slug = MEDIA_TYPE_SLUG[type];
  const title = hasMedia ? media.Name : t("wanted.mediaId", { id: item.MediaId });
  const releaseOrData = item.ReleaseTitle || item.Data;

  return (
    <li className="flex items-start gap-3 px-4 py-3">
      <div
        className={`w-12 shrink-0 overflow-hidden rounded-md border border-border bg-muted ${posterAspect(type)}`}
      >
        <CoverImg
          src={media?.Cover ?? ""}
          alt={title}
          className="h-full w-full object-cover"
        />
      </div>

      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          {hasMedia && slug ? (
            <Link
              to={`/${slug}/${media.Id}`}
              className="truncate font-medium text-primary hover:underline"
              title={title}
            >
              {title}
            </Link>
          ) : (
            <span className="truncate font-medium" title={title}>
              {title}
            </span>
          )}
          {hasMedia && media?.Type != null && <MediaTypeBadge type={media.Type} />}
        </div>
        {item.PartLabel && (
          <p className="text-sm text-muted-foreground">{item.PartLabel}</p>
        )}
        <p
          className="line-clamp-2 font-mono text-xs text-muted-foreground"
          title={releaseOrData || undefined}
        >
          {releaseOrData || t("common.dash")}
        </p>
        {item.ReleaseTitle && item.Data && item.Data !== item.ReleaseTitle && (
          <p
            className="line-clamp-1 font-mono text-xs text-muted-foreground/80"
            title={item.Data}
          >
            {item.Data}
          </p>
        )}
        <p className="text-xs text-muted-foreground">
          {new Date(item.CreatedAt).toLocaleString()}
        </p>
      </div>

      <div className="flex shrink-0 flex-col items-end gap-1.5">
        <HistoryEventBadge event={item.EventType} />
      </div>
    </li>
  );
}

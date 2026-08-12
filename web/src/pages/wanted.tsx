import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { QueueItem, WantedItem } from "@/lib/api/types";
import { MEDIA_TYPE_SLUG, MediaType } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CoverImg } from "@/components/ui/cover-img";
import { Spinner } from "@/components/ui/spinner";
import { MediaTypeBadge } from "@/components/media-type-badge";
import { ReleasePickerDialog } from "@/components/release-picker-dialog";
import { cn } from "@/lib/cn";

interface PickerState {
  media: WantedItem["Media"];
  season?: number;
  episode?: number;
  partIds?: number[];
}

type TypeFilter = "all" | MediaType;

const TYPE_FILTERS: { value: TypeFilter; slug?: string }[] = [
  { value: "all" },
  { value: MediaType.Movie, slug: "movies" },
  { value: MediaType.Series, slug: "series" },
  { value: MediaType.Book, slug: "books" },
  { value: MediaType.MusicAlbum, slug: "music" },
];

const STATE_BADGE: Record<string, "default" | "secondary" | "success" | "destructive" | "info"> = {
  queued: "secondary",
  running: "info",
  completed: "success",
  failed: "destructive",
  canceled: "secondary",
};

function formatEpisodeLabel(item: WantedItem): string | null {
  const { Part: part, Media: media, Season: season } = item;
  if (media.Type !== MediaType.Series) {
    return part.Name;
  }
  const ep = part.GroupOrder;
  if (season == null && ep == null && !part.Name) return null;
  const code =
    season != null && ep != null
      ? `S${String(season).padStart(2, "0")}E${String(ep).padStart(2, "0")}`
      : ep != null
        ? `E${String(ep).padStart(2, "0")}`
        : null;
  if (code && part.Name) return `${code} · ${part.Name}`;
  if (code) return code;
  return part.Name;
}

function findActiveGrab(queue: QueueItem[] | undefined, item: WantedItem): QueueItem | undefined {
  if (!queue?.length) return undefined;
  return queue.find((q) => {
    if (q.State !== "queued" && q.State !== "running") return false;
    if (q.MediaId !== item.Media.Id) return false;
    if (!q.PartIds || q.PartIds.length === 0) return true;
    return q.PartIds.includes(item.Part.Id);
  });
}

export function WantedPage() {
  const { t } = useTranslation();
  const [picker, setPicker] = useState<PickerState | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const queryClient = useQueryClient();

  const wantedQuery = useQuery({
    queryKey: ["parts/wanted"],
    queryFn: () => api.get<WantedItem[]>("/api/parts/wanted?page=1&limit=100"),
    refetchInterval: 15_000,
  });

  const queueQuery = useQuery({
    queryKey: ["queue", "active-match"],
    queryFn: () => api.get<QueueItem[]>("/api/queue?page=1&limit=100"),
    refetchInterval: 5_000,
  });

  const counts = useMemo(() => {
    const items = wantedQuery.data ?? [];
    const byType: Record<number, number> = {};
    for (const item of items) {
      byType[item.Media.Type] = (byType[item.Media.Type] ?? 0) + 1;
    }
    return { total: items.length, byType };
  }, [wantedQuery.data]);

  const filtered = useMemo(() => {
    const items = wantedQuery.data ?? [];
    if (typeFilter === "all") return items;
    return items.filter((item) => item.Media.Type === typeFilter);
  }, [wantedQuery.data, typeFilter]);

  const openSearch = (item: WantedItem) => {
    const { Part: part, Media: media, Season: season } = item;
    if (media.Type === MediaType.Series) {
      setPicker({
        media,
        season: season ?? undefined,
        episode: part.GroupOrder ?? undefined,
        partIds: [part.Id],
      });
      setPickerOpen(true);
      return;
    }
    setPicker({ media, partIds: [part.Id] });
    setPickerOpen(true);
  };

  return (
    <div className="mx-auto max-w-6xl px-8 py-8">
      <header className="mb-6">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t("wanted.title")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("wanted.description")}
            </p>
          </div>
          {!wantedQuery.isLoading && (wantedQuery.data?.length ?? 0) > 0 && (
            <p className="text-sm text-muted-foreground">
              {t("wanted.count", { count: counts.total })}
            </p>
          )}
        </div>

        {!wantedQuery.isLoading && (wantedQuery.data?.length ?? 0) > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            {TYPE_FILTERS.map((f) => {
              const count =
                f.value === "all" ? counts.total : (counts.byType[f.value] ?? 0);
              const active = typeFilter === f.value;
              const label =
                f.value === "all"
                  ? t("wanted.filterAll")
                  : t(`mediaTypeNav.${f.slug}`);
              return (
                <button
                  key={String(f.value)}
                  type="button"
                  onClick={() => setTypeFilter(f.value)}
                  className={cn(
                    "inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors",
                    active
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border bg-background text-muted-foreground hover:bg-muted/50"
                  )}
                >
                  {label}
                  <span
                    className={cn(
                      "rounded-sm px-1.5 py-0.5 tabular-nums",
                      active ? "bg-primary/15" : "bg-muted"
                    )}
                  >
                    {count}
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </header>

      {wantedQuery.isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner /> {t("common.loading")}
        </div>
      )}
      {wantedQuery.isError && (
        <p className="text-sm text-destructive">
          {(wantedQuery.error as Error).message}
        </p>
      )}

      {wantedQuery.data && wantedQuery.data.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
          <p className="text-sm text-muted-foreground">{t("wanted.empty")}</p>
        </div>
      )}

      {wantedQuery.data && wantedQuery.data.length > 0 && filtered.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-16 text-center">
          <p className="text-sm text-muted-foreground">{t("wanted.empty")}</p>
        </div>
      )}

      {filtered.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-border bg-popover">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
              <tr>
                <th className="px-4 py-2 font-medium">{t("wanted.table.media")}</th>
                <th className="px-4 py-2 font-medium">{t("wanted.table.type")}</th>
                <th className="px-4 py-2 font-medium">{t("wanted.table.episode")}</th>
                <th className="px-4 py-2 font-medium">{t("wanted.table.status")}</th>
                <th className="px-4 py-2 text-right font-medium">{t("wanted.table.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filtered.map((item) => {
                const grab = findActiveGrab(queueQuery.data, item);
                const episode = formatEpisodeLabel(item);
                const slug = MEDIA_TYPE_SLUG[item.Media.Type];
                return (
                  <tr key={item.Part.Id}>
                    <td className="max-w-sm px-4 py-2.5">
                      <div className="flex min-w-0 items-center gap-3">
                        <div className="h-14 w-10 shrink-0 overflow-hidden rounded-md border border-border bg-muted">
                          <CoverImg
                              src={item.Media.Cover ?? ""}
                              alt={item.Media.Name}
                              className="h-full w-full object-cover"
                            />
                        </div>
                        <div className="min-w-0 flex-1">
                          <Link
                            to={`/${slug}/${item.Media.Id}`}
                            className="block truncate font-medium text-primary hover:underline"
                            title={item.Media.Name}
                          >
                            {item.Media.Name}
                          </Link>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-2.5">
                      <MediaTypeBadge type={item.Media.Type} />
                    </td>
                    <td className="max-w-xs px-4 py-2.5 text-muted-foreground">
                      <span className="line-clamp-2" title={episode ?? undefined}>
                        {episode ?? t("common.dash")}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      {grab ? (
                        <div className="flex flex-col gap-1.5">
                          <Badge variant={STATE_BADGE[grab.State] ?? "secondary"}>
                            {grab.State === "running"
                              ? t("wanted.status.downloading")
                              : t("wanted.status.queued")}
                          </Badge>
                          {grab.State === "running" && (
                            <div className="flex items-center gap-2">
                              <div className="h-1.5 w-16 overflow-hidden rounded bg-muted">
                                <div
                                  className="h-full bg-primary transition-all"
                                  style={{
                                    width: `${Math.round(grab.Progress * 100)}%`,
                                  }}
                                />
                              </div>
                              <span className="text-xs tabular-nums text-muted-foreground">
                                {Math.round(grab.Progress * 100)}%
                              </span>
                            </div>
                          )}
                        </div>
                      ) : (
                        <Badge variant="secondary">{t("wanted.status.missing")}</Badge>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={!!grab}
                        onClick={() => openSearch(item)}
                      >
                        <Search />
                        {t("actions.searchButton")}
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {picker && (
        <ReleasePickerDialog
          media={picker.media}
          open={pickerOpen}
          onOpenChange={(o) => {
            setPickerOpen(o);
            if (!o) {
              queryClient.invalidateQueries({ queryKey: ["parts/wanted"] });
              queryClient.invalidateQueries({ queryKey: ["queue"] });
            }
          }}
          season={picker.season}
          episode={picker.episode}
          partIds={picker.partIds}
        />
      )}
    </div>
  );
}

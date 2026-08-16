import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";
import { api } from "@/lib/api/client";
import type { QueueListItem } from "@/lib/api/types";
import { MEDIA_TYPE_SLUG, MediaType } from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CoverImg } from "@/components/ui/cover-img";
import { Spinner } from "@/components/ui/spinner";
import { MediaTypeBadge } from "@/components/media-type-badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const PAGE_SIZE = 20;

const STATE_BADGE: Record<string, "default" | "secondary" | "success" | "destructive" | "info"> = {
  queued: "secondary",
  running: "info",
  completed: "success",
  failed: "destructive",
  canceled: "secondary",
};

function posterAspect(type: MediaType): string {
  if (type === MediaType.MusicAlbum) return "aspect-square";
  return type === MediaType.Movie || type === MediaType.Series
    ? "aspect-[2/3]"
    : "aspect-[3/4]";
}

export function QueuePage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const sentinelRef = useRef<HTMLDivElement>(null);
  const [removeId, setRemoveId] = useState<number | null>(null);

  const queueQuery = useInfiniteQuery({
    queryKey: ["queue", "infinite"],
    queryFn: ({ pageParam }) =>
      api.get<QueueListItem[]>(
        `/api/queue?page=${pageParam}&limit=${PAGE_SIZE}`
      ),
    initialPageParam: 1,
    getNextPageParam: (last, pages) =>
      last.length < PAGE_SIZE ? undefined : pages.length + 1,
    refetchInterval: 5000,
  });

  const items = useMemo(
    () => (queueQuery.data?.pages ?? []).flat(),
    [queueQuery.data]
  );

  const remove = useMutation({
    mutationFn: (id: number) => api.del(`/api/queue/${id}`),
    onSuccess: () => {
      setRemoveId(null);
      queryClient.invalidateQueries({ queryKey: ["queue"] });
      queryClient.invalidateQueries({ queryKey: ["history"] });
      queryClient.invalidateQueries({ queryKey: ["parts/wanted"] });
      toast.success(t("queue.removed"));
    },
    onError: (e) => toast.error((e as Error).message),
  });

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target) return;

    const obs = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting) return;
        if (queueQuery.hasNextPage && !queueQuery.isFetchingNextPage) {
          void queueQuery.fetchNextPage();
        }
      },
      { root: null, rootMargin: "120px", threshold: 0 }
    );
    obs.observe(target);
    return () => obs.disconnect();
  }, [
    queueQuery.hasNextPage,
    queueQuery.isFetchingNextPage,
    queueQuery.fetchNextPage,
    items.length,
  ]);

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <header className="mb-8">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{t("queue.title")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("queue.description")}
            </p>
          </div>
          {!queueQuery.isLoading && items.length > 0 && (
            <p className="text-sm text-muted-foreground">
              {t("queue.count", { count: items.length })}
            </p>
          )}
        </div>
      </header>

      {queueQuery.isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner /> {t("common.loading")}
        </div>
      )}
      {queueQuery.isError && (
        <p className="text-sm text-destructive">
          {(queueQuery.error as Error).message}
        </p>
      )}

      {!queueQuery.isLoading && items.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
          <p className="text-sm text-muted-foreground">{t("empty.queueEmpty")}</p>
        </div>
      )}

      {items.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-border bg-popover">
          <ul className="divide-y divide-border">
            {items.map((q) => (
              <QueueRow key={q.Id} item={q} onRemove={() => setRemoveId(q.Id)} />
            ))}
          </ul>
          <div ref={sentinelRef} className="h-1" />
          {queueQuery.isFetchingNextPage && (
            <div className="flex items-center gap-2 px-4 py-3 text-sm text-muted-foreground">
              <Spinner /> {t("search.loadingMore")}
            </div>
          )}
        </div>
      )}

      <Dialog
        open={removeId != null}
        onOpenChange={(open) => {
          if (!open && !remove.isPending) setRemoveId(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("queue.removeConfirmTitle")}</DialogTitle>
            <DialogDescription>{t("queue.removeConfirmDescription")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setRemoveId(null)}
              disabled={remove.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (removeId != null) remove.mutate(removeId);
              }}
              disabled={remove.isPending}
            >
              {remove.isPending ? t("queue.removing") : t("queue.remove")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function QueueRow({
  item,
  onRemove,
}: {
  item: QueueListItem;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const media = item.Media;
  const hasMedia = media?.Id > 0 && !!media.Name;
  const type = media?.Type ?? MediaType.Movie;
  const slug = MEDIA_TYPE_SLUG[type];
  const title = hasMedia ? media.Name : t("wanted.mediaId", { id: item.MediaId });
  const running = item.State === "running";

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
          title={item.ReleaseTitle || undefined}
        >
          {item.ReleaseTitle || t("common.dash")}
        </p>
        <p className="text-xs text-muted-foreground">
          {item.GrabberName}
          {" · "}
          {new Date(item.AddedAt).toLocaleString()}
        </p>
      </div>

      <div className="flex shrink-0 items-start gap-1">
        <div className="flex flex-col items-end gap-1.5">
          <Badge variant={STATE_BADGE[item.State] ?? "secondary"}>
            {t(`badges.${item.State}`)}
          </Badge>
          {running && (
            <div className="flex items-center gap-2">
              <div className="h-1.5 w-16 overflow-hidden rounded bg-muted">
                <div
                  className="h-full bg-primary transition-all"
                  style={{ width: `${Math.round(item.Progress * 100)}%` }}
                />
              </div>
              <span className="w-8 text-right text-xs tabular-nums text-muted-foreground">
                {Math.round(item.Progress * 100)}%
              </span>
            </div>
          )}
        </div>
        <Button
          size="icon"
          variant="ghost"
          onClick={onRemove}
          title={t("queue.remove")}
        >
          <Trash2 className="size-4" />
        </Button>
      </div>
    </li>
  );
}

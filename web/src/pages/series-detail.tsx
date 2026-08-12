import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { ChevronDown, ChevronRight, RefreshCw, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Media, Part, PartGroup, History } from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ReleasePickerDialog } from "@/components/release-picker-dialog";
import { MediaProfilePicker } from "@/components/media-profile-picker";
import { HistoryEventBadge } from "@/components/history-event-badge";
import { DeleteMediaButton } from "@/components/delete-media-button";

interface PickerTarget {
  season: number;
  episode?: number;
  partIds: number[];
}

export function SeriesDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const mediaId = Number(id);
  const queryClient = useQueryClient();
  const [collapsed, setCollapsed] = useState<Set<number>>(new Set());
  const [picker, setPicker] = useState<PickerTarget | null>(null);

  const mediaQuery = useQuery({
    queryKey: ["media", mediaId],
    queryFn: () => api.get<Media>(`/api/media/${mediaId}`),
    enabled: Number.isFinite(mediaId),
  });

  const partsQuery = useQuery({
    queryKey: ["parts", mediaId],
    queryFn: () => api.get<Part[]>(`/api/parts/media/${mediaId}`),
    enabled: Number.isFinite(mediaId),
  });

  const groupsQuery = useQuery({
    queryKey: ["groups", mediaId],
    queryFn: () => api.get<PartGroup[]>(`/api/media/${mediaId}/groups`),
    enabled: Number.isFinite(mediaId),
  });

  const historyQuery = useQuery({
    queryKey: ["history", mediaId],
    queryFn: () =>
      api.get<History[]>(`/api/history?media_id=${mediaId}&limit=20`),
    enabled: Number.isFinite(mediaId),
  });

  const refresh = useMutation({
    mutationFn: () => api.post<Media>(`/api/media/${mediaId}/refresh`),
    onSuccess: (m) => {
      queryClient.setQueryData(["media", mediaId], m);
      queryClient.invalidateQueries({ queryKey: ["parts", mediaId] });
      queryClient.invalidateQueries({ queryKey: ["groups", mediaId] });
      toast.success(t("messages.metadataRefreshed"));
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const toggleMonitor = useMutation({
    mutationFn: (p: Part) =>
      api.put<Part>(`/api/parts/${p.Id}`, { monitored: !p.Monitored }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["parts", mediaId] });
      queryClient.invalidateQueries({ queryKey: ["parts/wanted"] });
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const seasons = useMemo(() => {
    const groups = (groupsQuery.data ?? []).slice().sort((a, b) => a.Order - b.Order);
    const byGroup = new Map<number, Part[]>();
    for (const p of partsQuery.data ?? []) {
      if (p.GroupId == null) continue;
      const list = byGroup.get(p.GroupId) ?? [];
      list.push(p);
      byGroup.set(p.GroupId, list);
    }
    return groups.map((g) => ({
      group: g,
      episodes: (byGroup.get(g.Id) ?? []).slice().sort(
        (a, b) => (a.GroupOrder ?? 0) - (b.GroupOrder ?? 0)
      ),
    }));
  }, [groupsQuery.data, partsQuery.data]);

  if (mediaQuery.isLoading) {
    return <div className="p-8 text-sm text-muted-foreground">{t("common.loading")}</div>;
  }
  if (mediaQuery.error || !mediaQuery.data) {
    return (
      <div className="p-8 text-sm text-destructive">
        {(mediaQuery.error as Error)?.message ?? t("common.notFound")}
      </div>
    );
  }

  const media = mediaQuery.data;
  const parts = partsQuery.data ?? [];
  const importedCount = parts.filter((p) => p.Path != null).length;
  const wantedCount = parts.filter((p) => p.Monitored && p.Path == null).length;

  const toggleCollapse = (groupId: number) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) next.delete(groupId);
      else next.add(groupId);
      return next;
    });
  };

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <div className="flex gap-8">
        <div className="w-56 shrink-0">
          <div className="aspect-[2/3] overflow-hidden rounded-lg border border-border bg-muted">
            {media.Cover ? (
              <img src={media.Cover} alt={media.Name} className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
                {t("common.noCover")}
              </div>
            )}
          </div>
        </div>

        <div className="flex-1 space-y-6">
          <div className="flex items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-semibold tracking-tight">{media.Name}</h1>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                {importedCount > 0 && <Badge variant="success">{importedCount} {t("badges.imported")}</Badge>}
                {wantedCount > 0 && <Badge variant="warning">{wantedCount} {t("badges.wantedBadge")}</Badge>}
              </div>
              <div className="mt-3">
                <MediaProfilePicker media={media} />
              </div>
            </div>
            <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => refresh.mutate()}
                disabled={refresh.isPending}
              >
                <RefreshCw className={refresh.isPending ? "animate-spin" : ""} />
                {t("actions.refresh")}
              </Button>
              <DeleteMediaButton media={media} />
            </div>
          </div>

          <section className="space-y-4">
            {seasons.length === 0 && (
              <p className="text-sm text-muted-foreground">{t("empty.noSeasons")}</p>
            )}
            {seasons.map(({ group, episodes }) => {
              const isCollapsed = collapsed.has(group.Id);
              const seasonWanted = episodes.filter((e) => e.Monitored && e.Path == null);
              const seasonImported = episodes.filter((e) => e.Path != null).length;
              return (
                <div key={group.Id} className="overflow-hidden rounded-lg border border-border">
                  <div className="flex items-center justify-between bg-muted/50 px-4 py-3">
                    <button
                      onClick={() => toggleCollapse(group.Id)}
                      className="flex items-center gap-2 text-sm font-medium"
                    >
                      {isCollapsed ? <ChevronRight className="size-4" /> : <ChevronDown className="size-4" />}
                      {group.Name}
                      <span className="text-xs text-muted-foreground">
                        {seasonImported}/{episodes.length}
                      </span>
                    </button>
                    {seasonWanted.length > 0 && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() =>
                          setPicker({
                            season: group.Order,
                            partIds: seasonWanted.map((e) => e.Id),
                          })
                        }
                      >
                        <Search />
                        {t("actions.searchSeason")}
                      </Button>
                    )}
                  </div>
                  {!isCollapsed && (
                    <table className="w-full text-sm">
                      <tbody className="divide-y divide-border">
                        {episodes.map((ep) => (
                          <tr key={ep.Id}>
                            <td className="w-12 px-4 py-2.5 font-mono text-xs text-muted-foreground">
                              {ep.GroupOrder ?? t("common.dash")}
                            </td>
                            <td className="px-4 py-2.5">
                              <span className="font-medium">{ep.Name ?? t("episode.title", { n: ep.GroupOrder ?? "" })}</span>
                            </td>
                            <td className="px-4 py-2.5">
                              {ep.Path ? (
                                <Badge variant="success">{t("badges.imported")}</Badge>
                              ) : ep.Monitored ? (
                                <Badge variant="warning">{t("badges.wanted")}</Badge>
                              ) : (
                                <Badge variant="secondary">{t("badges.unmonitored")}</Badge>
                              )}
                            </td>
                            <td className="px-4 py-2.5 text-right">
                              <div className="flex items-center justify-end gap-2">
                                <Button
                                  size="sm"
                                  variant="ghost"
                                  disabled={toggleMonitor.isPending}
                                  onClick={() => toggleMonitor.mutate(ep)}
                                >
                                  {ep.Monitored ? t("actions.unmonitor") : t("actions.monitor")}
                                </Button>
                                {ep.Monitored && ep.Path == null && ep.GroupOrder != null && (
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() =>
                                      setPicker({
                                        season: group.Order,
                                        episode: ep.GroupOrder ?? undefined,
                                        partIds: [ep.Id],
                                      })
                                    }
                                  >
                                    <Search />
                                    {t("actions.searchButton")}
                                  </Button>
                                )}
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              );
            })}
          </section>

          <section>
            <h2 className="mb-2 text-sm font-medium uppercase tracking-wider text-muted-foreground">
              {t("detail.history")}
            </h2>
            {historyQuery.data && historyQuery.data.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("empty.noActivity")}</p>
            ) : (
              <div className="space-y-2">
                {(historyQuery.data ?? []).map((h) => (
                  <div
                    key={h.Id}
                    className="flex items-center justify-between rounded-lg border border-border px-4 py-2.5 text-sm"
                  >
                    <div className="min-w-0">
                      <p className="truncate">{h.ReleaseTitle || h.Data}</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(h.CreatedAt).toLocaleString()}
                      </p>
                    </div>
                    <HistoryEventBadge event={h.EventType} />
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>
      </div>

      <ReleasePickerDialog
        media={media}
        open={picker !== null}
        onOpenChange={(o) => !o && setPicker(null)}
        season={picker?.season}
        episode={picker?.episode}
        partIds={picker?.partIds}
      />
    </div>
  );
}

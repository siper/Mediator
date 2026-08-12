import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { RefreshCw, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Media, Part, History } from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ReleasePickerDialog } from "@/components/release-picker-dialog";
import { MediaProfilePicker } from "@/components/media-profile-picker";
import { HistoryEventBadge } from "@/components/history-event-badge";
import { DeleteMediaButton } from "@/components/delete-media-button";

export function MusicDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const mediaId = Number(id);
  const queryClient = useQueryClient();
  const [pickerOpen, setPickerOpen] = useState(false);

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
      toast.success(t("messages.metadataRefreshed"));
    },
    onError: (e) => toast.error((e as Error).message),
  });

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
  const imported = parts.some((p) => p.Path != null);
  const wanted = parts.some((p) => p.Path == null && p.Monitored);

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <div className="flex gap-8">
        <div className="w-56 shrink-0">
          <div className="aspect-square overflow-hidden rounded-lg border border-border bg-muted">
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
                {imported ? (
                  <Badge variant="success">{t("badges.importedBadge")}</Badge>
                ) : wanted ? (
                  <Badge variant="warning">{t("badges.wantedBadge")}</Badge>
                ) : (
                  <Badge variant="secondary">{t("badges.noPartsBadge")}</Badge>
                )}
                <Badge variant="outline">{media.Status || t("badges.completed")}</Badge>
                <Badge variant="secondary">{media.ProviderID}</Badge>
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
              <Button size="sm" onClick={() => setPickerOpen(true)}>
                <Search />
                {t("actions.searchReleases")}
              </Button>
              <DeleteMediaButton media={media} />
            </div>
          </div>

          <section>
            <h2 className="mb-2 text-sm font-medium uppercase tracking-wider text-muted-foreground">
              {t("detail.files")}
            </h2>
            {parts.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("empty.noParts")}</p>
            ) : (
              <div className="overflow-hidden rounded-lg border border-border">
                <table className="w-full text-sm">
                  <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
                    <tr>
                      <th className="px-4 py-2 font-medium">{t("detail.part")}</th>
                      <th className="px-4 py-2 font-medium">{t("detail.path")}</th>
                      <th className="px-4 py-2 font-medium">{t("detail.status")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {parts.map((p) => (
                      <tr key={p.Id}>
                        <td className="px-4 py-2.5">
                          {p.Name ? (
                            <span>{p.Name}</span>
                          ) : (
                            <span className="font-mono text-xs text-muted-foreground">
                              #{p.Id}
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-2.5 font-mono text-xs">{p.Path ?? t("common.dash")}</td>
                        <td className="px-4 py-2.5">
                          {p.Path ? (
                            <Badge variant="success">{t("badges.imported")}</Badge>
                          ) : p.Monitored ? (
                            <Badge variant="warning">{t("badges.wanted")}</Badge>
                          ) : (
                            <Badge variant="secondary">{t("badges.unmonitored")}</Badge>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
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
      {mediaQuery.data && (
        <ReleasePickerDialog
          media={mediaQuery.data}
          open={pickerOpen}
          onOpenChange={setPickerOpen}
        />
      )}
    </div>
  );
}

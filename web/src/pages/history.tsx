import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { History, Media } from "@/lib/api/types";
import { MediaType } from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";

function eventVariant(event: string): "default" | "secondary" | "success" | "destructive" | "info" {
  switch (event) {
    case "grabbed":
      return "info";
    case "imported":
      return "success";
    case "failed":
      return "destructive";
    default:
      return "secondary";
  }
}

export function HistoryPage() {
  const { t } = useTranslation();
  const historyQuery = useQuery({
    queryKey: ["history"],
    queryFn: () => api.get<History[]>("/api/history?page=1&limit=100"),
  });

  const moviesQuery = useQuery({
    queryKey: ["media", "by-type", MediaType.Movie],
    queryFn: () => api.get<Media[]>(`/api/media?type=${MediaType.Movie}&limit=100`),
  });

  const byId = new Map((moviesQuery.data ?? []).map((m) => [m.Id, m]));

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <header className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight">{t("history.title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("history.description")}
        </p>
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

      {historyQuery.data && historyQuery.data.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
          <p className="text-sm text-muted-foreground">{t("empty.noActivity")}</p>
        </div>
      )}

      {historyQuery.data && historyQuery.data.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
              <tr>
                <th className="px-4 py-2 font-medium">{t("history.media")}</th>
                <th className="px-4 py-2 font-medium">{t("history.event")}</th>
                <th className="px-4 py-2 font-medium">{t("history.releaseFile")}</th>
                <th className="px-4 py-2 font-medium">{t("history.when")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {historyQuery.data.map((h) => (
                <tr key={h.Id}>
                  <td className="px-4 py-2.5 font-medium">
                    {byId.get(h.MediaId)?.Name ?? `#${h.MediaId}`}
                  </td>
                  <td className="px-4 py-2.5">
                    <Badge variant={eventVariant(h.EventType)}>{t(`badges.${h.EventType}`)}</Badge>
                  </td>
                  <td className="max-w-md truncate px-4 py-2.5 font-mono text-xs">
                    {h.ReleaseTitle || h.Data || t("common.dash")}
                  </td>
                  <td className="px-4 py-2.5 text-xs text-muted-foreground">
                    {new Date(h.CreatedAt).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Search, Loader2, Download } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Media, ScoredRelease } from "@/lib/api/types";
import { MediaType } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

function formatSize(bytes: number, units: string[]): string {
  if (!bytes) return "—";
  let i = 0;
  let v = bytes;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(v >= 100 || i === 0 ? 0 : 1)} ${units[i]}`;
}

function defaultReleaseQuery(media: Media): string {
  const original = (media.OriginalName ?? "").trim();
  const name = (media.Name ?? "").trim();
  if (media.Type === MediaType.Book) {
    const sep = " — ";
    const i = name.indexOf(sep);
    if (i >= 0) {
      const title = name.slice(i + sep.length).trim();
      if (title) return title;
    }
  }
  if (name) return name;
  return original;
}

export function ReleasePickerDialog({
  media,
  open,
  onOpenChange,
  season,
  episode,
  partIds,
}: {
  media: Media;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  season?: number;
  episode?: number;
  partIds?: number[];
}) {
  const { t } = useTranslation();
  const units: string[] = t("releasePicker.units", { returnObjects: true }) as string[];
  const [query, setQuery] = useState(() => defaultReleaseQuery(media));
  const queryClient = useQueryClient();

  useEffect(() => {
    setQuery(defaultReleaseQuery(media));
  }, [media.Id, media.Name, media.OriginalName, media.Type]);

  const searchQuery = useQuery({
    queryKey: ["releases/search", media.Id, media.Type, query, season, episode],
    queryFn: () =>
      api.post<ScoredRelease[]>("/api/releases/search", {
        query,
        media_id: media.Id,
        type: media.Type,
        quality_profile_id: media.QualityProfileID ?? null,
        season,
        episode,
      }),
    enabled: open && !!query && query.trim().length > 0,
  });

  const grab = useMutation({
    mutationFn: (r: ScoredRelease) =>
      api.post("/api/releases/grab", {
        media_id: media.Id,
        release: r.Release,
        part_ids: partIds,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["queue"] });
      queryClient.invalidateQueries({ queryKey: ["history"] });
      queryClient.invalidateQueries({ queryKey: ["parts", media.Id] });
      queryClient.invalidateQueries({ queryKey: ["parts/wanted"] });
      toast.success(t("messages.grabbed"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("releasePicker.title")}</DialogTitle>
          <DialogDescription>{media.Name}</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              value={query ?? ""}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && searchQuery.refetch()}
            />
          </div>
          <Button
            variant="outline"
            onClick={() => searchQuery.refetch()}
            disabled={searchQuery.isFetching}
          >
            {searchQuery.isFetching ? <Loader2 className="animate-spin" /> : <Search />}
            {t("releasePicker.search")}
          </Button>
        </div>

        <div className="max-h-96 space-y-2 overflow-y-auto">
          {searchQuery.isLoading && (
            <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" /> {t("releasePicker.searching")}
            </div>
          )}
          {searchQuery.isError && (
            <p className="text-sm text-destructive">
              {(searchQuery.error as Error).message}
            </p>
          )}
          {searchQuery.data && searchQuery.data.length === 0 && (
            <p className="py-6 text-sm text-muted-foreground">{t("releasePicker.noResults")}</p>
          )}
          {(searchQuery.data ?? []).map((r, i) => (
            <div
              key={i}
              className="flex items-center gap-3 rounded-lg border border-border p-3"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{r.Release.Title}</p>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <span className="font-mono">{r.Release.Indexer}</span>
                  <span>{formatSize(r.Release.Size, units)}</span>
                  {r.Release.Seeders > 0 && (
                    <span className="text-success">{t("releasePicker.seeders", { seeders: r.Release.Seeders })}</span>
                  )}
                  {r.Parsed?.Quality?.Name && (
                    <Badge variant="secondary" className="font-mono text-[10px]">
                      {r.Parsed.Quality.Name}
                    </Badge>
                  )}
                </div>
              </div>
              <Button
                size="sm"
                onClick={() => grab.mutate(r)}
                disabled={grab.isPending}
              >
                <Download />
                {t("actions.grab")}
              </Button>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}

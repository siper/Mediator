import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Media, QualityProfile } from "@/lib/api/types";
import { MediaType } from "@/lib/api/types";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export function MediaProfilePicker({ media }: { media: Media }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const profilesQuery = useQuery({
    queryKey: ["profiles", media.Type],
    queryFn: () => api.get<QualityProfile[]>("/api/quality-profiles"),
  });

  const available = (profilesQuery.data ?? []).filter((p) => p.Type === media.Type);

  const update = useMutation({
    mutationFn: (profileId: number | null) =>
      api.patch<Media>(`/api/media/${media.Id}/profile`, {
        quality_profile_id: profileId,
      }),
    onSuccess: (m) => {
      queryClient.setQueryData(["media", media.Id], m);
      toast.success(t("messages.qualityProfileUpdated"));
    },
    onError: (e) => toast.error((e as Error).message),
  });

  if (
    media.Type !== MediaType.Movie &&
    media.Type !== MediaType.Series &&
    media.Type !== MediaType.MusicAlbum
  ) {
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-muted-foreground">{t("profiles.qualityProfileLabel")}</span>
      <Select
        value={media.QualityProfileID ? String(media.QualityProfileID) : "__none__"}
        onValueChange={(v) => update.mutate(v === "__none__" ? null : Number(v))}
        disabled={update.isPending}
      >
        <SelectTrigger className="w-48">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__none__">{t("common.none")}</SelectItem>
          {available.map((p) => (
            <SelectItem key={p.Id} value={String(p.Id)}>
              {p.Name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

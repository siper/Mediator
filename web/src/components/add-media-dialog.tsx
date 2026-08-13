import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Library, Media, MediaRequest, PublicSettings, QualityProfile, SearchResult } from "@/lib/api/types";
import { MEDIA_TYPE_KEY, MediaType } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { CoverImg } from "@/components/ui/cover-img";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { useUIStore } from "@/store/ui";
import { useAuthStore } from "@/store/auth";

export function AddMediaDialog() {
  const result = useUIStore((s) => s.addResult);
  const setResult = useUIStore((s) => s.setAddResult);
  const open = result !== null;

  return (
    <Dialog open={open} onOpenChange={(o) => !o && setResult(null)}>
      <DialogContent className="max-w-lg">
        {result && <AddBody result={result} onClose={() => setResult(null)} />}
      </DialogContent>
    </Dialog>
  );
}

function AddBody({ result, onClose }: { result: SearchResult; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [folder, setFolder] = useState("");
  const [searchMissing, setSearchMissing] = useState(true);

  const { user } = useAuthStore();
  const isAdmin = user?.Role === "admin";

  const publicSettingsQuery = useQuery({
    queryKey: ["settings", "public"],
    queryFn: () => api.get<PublicSettings>("/api/settings/public"),
  });
  const requestsEnabled = publicSettingsQuery.data?.requests_enabled === "true";
  const shouldRequest = requestsEnabled && !isAdmin;

  const librariesQuery = useQuery({
    queryKey: ["libraries"],
    queryFn: () => api.get<Library[]>("/api/libraries"),
  });

  const profilesQuery = useQuery({
    queryKey: ["profiles", result.MediaType],
    queryFn: () => api.get<QualityProfile[]>("/api/quality-profiles"),
  });

  const availableProfiles = (profilesQuery.data ?? []).filter(
    (p) => p.Type === result.MediaType
  );

  const canHaveProfile =
    result.MediaType === MediaType.Movie ||
    result.MediaType === MediaType.Series ||
    result.MediaType === MediaType.MusicAlbum;

  useEffect(() => {
    if (availableProfiles.length > 0 && profileId === "") {
      setProfileId(String(availableProfiles[0].Id));
    }
  }, [availableProfiles, profileId]);

  const available = (librariesQuery.data ?? []).filter(
    (l) => l.Type === result.MediaType
  );

  useEffect(() => {
    if (available.length > 0 && libraryId === "") {
      setLibraryId(String(available[0].Id));
    }
  }, [available, libraryId]);

  const importMedia = useMutation({
    mutationFn: () =>
      api.post<Media>("/api/providers/import", {
        provider: result.ProviderName,
        external_id: result.ExternalID,
         type: result.MediaType,
        library_id: Number(libraryId),
        quality_profile_id: profileId ? Number(profileId) : null,
        folder: folder.trim(),
        cover_url: result.CoverURL,
      }),
     onSuccess: (m) => {
      queryClient.invalidateQueries({ queryKey: ["media"] });
      toast.success(t("messages.added", { name: m.Name }));
      onClose();
      if (searchMissing) {
        grabMissing.mutate(m.Id);
      }
      const base =
        m.Type === MediaType.Movie ? "movies" :
        m.Type === MediaType.Series ? "series" :
        m.Type === MediaType.Book ? "books" :
        m.Type === MediaType.MusicAlbum ? "music" : null;
      if (base) navigate(`/${base}/${m.Id}`);
    },
     onError: (e) => toast.error((e as Error).message),
    });

  const grabMissing = useMutation({
    mutationFn: (mediaId: number) =>
      api.post<{ grabbed: number }>(`/api/media/${mediaId}/grab-missing`),
    onSuccess: (data) => {
      if (data.grabbed > 0) {
        toast.success(t("messages.searchedMissing", { count: data.grabbed }));
      }
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const createRequest = useMutation({
    mutationFn: () =>
      api.post<MediaRequest>("/api/requests", {
        provider: result.ProviderName,
        external_id: result.ExternalID,
        title: result.Title,
        cover: result.CoverURL,
        type: result.MediaType,
        library_id: Number(libraryId),
        quality_profile_id: profileId ? Number(profileId) : null,
        folder: folder.trim(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requests"] });
      toast.success(t("messages.requestSubmitted"));
      onClose();
      navigate("/requests");
    },
    onError: (e) => {
      const msg = (e as Error).message;
      if (msg.includes("already")) {
        toast(t("messages.alreadyRequested"));
      } else {
        toast.error(msg);
      }
    },
  });

  const submit = () => {
    if (!libraryId) {
      toast.error(t("messages.selectLibrary"));
      return;
    }
    if (canHaveProfile && !profileId) {
      toast.error(t("messages.selectProfile"));
      return;
    }
    if (shouldRequest) {
      createRequest.mutate();
      return;
    }
    importMedia.mutate();
  };

  const typeKey = MEDIA_TYPE_KEY[result.MediaType];

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("addMedia.title")}</DialogTitle>
        <DialogDescription>
          {t("addMedia.description", { type: t(`mediaType.${typeKey}`) })}
        </DialogDescription>
      </DialogHeader>

      <div className="flex overflow-hidden gap-4">
        <div className="h-32 w-24 shrink-0 overflow-hidden rounded-md border border-border bg-muted">
          <CoverImg
            src={result.CoverURL}
            alt={result.Title}
            className="h-full w-full object-cover"
          />
        </div>
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <p className="min-w-0 truncate font-medium">{result.Title}</p>
          <div className="flex items-center gap-2">
            <Badge variant="secondary">{t(`mediaType.${typeKey}`)}</Badge>
            <span className="text-xs text-muted-foreground">{result.ProviderName}</span>
          </div>
          <Select value={libraryId || undefined} onValueChange={setLibraryId}>
            <SelectTrigger>
              <SelectValue placeholder={t("addMedia.librarySelect")} />
            </SelectTrigger>
            <SelectContent>
              {available.map((l) => (
                <SelectItem key={l.Id} value={String(l.Id)}>
                  {l.Name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {canHaveProfile && (
            <Select value={profileId || undefined} onValueChange={setProfileId}>
              <SelectTrigger>
                <SelectValue placeholder={t("addMedia.profileSelect")} />
              </SelectTrigger>
              <SelectContent>
                {availableProfiles.map((p) => (
                  <SelectItem key={p.Id} value={String(p.Id)}>
                    {p.Name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <Input
            type="text"
            placeholder={t("addMedia.folderPlaceholder")}
            value={folder}
            onChange={(e) => setFolder(e.target.value)}
          />
          {!shouldRequest && (
            <label className="flex items-center gap-3 cursor-pointer">
              <Switch
                checked={searchMissing}
                onCheckedChange={setSearchMissing}
              />
              <span className="text-sm">{t(`addMedia.searchMissing.${typeKey}`)}</span>
            </label>
          )}
          {available.length === 0 && (
            <p className="text-xs text-muted-foreground">
              {t("addMedia.noLibrary", { type: t(`mediaType.${typeKey}`) })}
            </p>
          )}
          {canHaveProfile && availableProfiles.length === 0 && (
            <p className="text-xs text-muted-foreground">
              {t("addMedia.noProfile")}
            </p>
          )}
        </div>
      </div>

      <Button
        onClick={submit}
        disabled={(shouldRequest ? createRequest.isPending : importMedia.isPending) || available.length === 0 || (canHaveProfile && availableProfiles.length === 0)}
        className="w-full"
      >
        {(shouldRequest ? createRequest.isPending : importMedia.isPending) ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <Plus className="size-4" />
        )}
        {(shouldRequest ? createRequest.isPending : importMedia.isPending)
          ? t("addMedia.requestingButton")
          : shouldRequest
            ? t("addMedia.requestButton")
            : t("addMedia.importButton")}
      </Button>
    </>
  );
}

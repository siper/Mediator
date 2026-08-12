import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Library } from "@/lib/api/types";
import { MediaType, MEDIA_TYPE_SLUG } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";

const LIBRARY_TYPE_OPTIONS = [
  { value: String(MediaType.Movie), type: MediaType.Movie },
  { value: String(MediaType.Series), type: MediaType.Series },
  { value: String(MediaType.Book), type: MediaType.Book },
  { value: String(MediaType.MusicAlbum), type: MediaType.MusicAlbum },
];

function previewSeason(fmt: string, t: (key: string, args?: Record<string, unknown>) => string): string {
  const f = fmt.trim() || t("quality.season");
  if (!f.includes("%")) return f;
  return f.replace(/%(0*)(\d*)d/, (_, zero, width) => {
    const w = width ? Number(width) : 0;
    return zero || w ? String(1).padStart(w, "0") : "1";
  });
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  library?: Library | null;
}

export function LibraryDialog({ open, onOpenChange, library }: Props) {
  const { t } = useTranslation();
  const editing = Boolean(library);
  const [name, setName] = useState("");
  const [path, setPath] = useState("");
  const [type, setType] = useState(String(MediaType.Movie));
  const [seasonFormat, setSeasonFormat] = useState("");

  useEffect(() => {
    if (!open) return;
    if (library) {
      setName(library.Name);
      setPath(library.Path);
      setType(String(library.Type));
      setSeasonFormat(library.Settings?.season_format ?? "");
    } else {
      setName("");
      setPath("");
      setType(String(MediaType.Movie));
      setSeasonFormat("");
    }
  }, [open, library]);

  const typeLabel = LIBRARY_TYPE_OPTIONS.find((o) => o.value === type);
  const typeLabelText = typeLabel ? t(`mediaTypeNav.${MEDIA_TYPE_SLUG[typeLabel.type]}`) : t("library.fallbackType");
  const resolvedName = name.trim() || typeLabelText;

  const queryClient = useQueryClient();

  const save = useMutation({
    mutationFn: () => {
      const body = {
        Name: resolvedName,
        Path: path,
        Type: Number(type),
        Settings: Number(type) === MediaType.Series ? { season_format: seasonFormat } : {},
      };
      if (library) {
        return api.put(`/api/libraries/${library.Id}`, body);
      }
      return api.post("/api/libraries", body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["libraries"] });
      queryClient.invalidateQueries({ queryKey: ["media"] });
      toast.success(library ? t("messages.libraryUpdated") : t("messages.libraryAdded"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = path.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{editing ? t("library.titleEdit") : t("library.titleAdd")}</DialogTitle>
          <DialogDescription>
            {editing
              ? t("library.descriptionEdit")
              : t("library.descriptionAdd")}
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2">
          <Badge variant="secondary">{typeLabelText}</Badge>
        </div>

        <div className="space-y-3">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("library.name")}</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={typeLabelText} />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("library.path")}</label>
            <Input
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder={t("library.placeholder")}
              className="font-mono text-xs"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("library.type")}</label>
            <Select value={type} onValueChange={setType}>
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LIBRARY_TYPE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {t(`mediaTypeNav.${MEDIA_TYPE_SLUG[o.type]}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {Number(type) === MediaType.Series && (
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">{t("library.seasonFormat")}</label>
              <Input
                value={seasonFormat}
                onChange={(e) => setSeasonFormat(e.target.value)}
                placeholder={t("quality.season")}
                className="font-mono text-xs"
              />
              <p className="text-xs text-muted-foreground">
                {t("library.seasonFormatHelp", { preview: previewSeason(seasonFormat, t) })}
              </p>
            </div>
          )}
        </div>

        <Button onClick={() => save.mutate()} disabled={!valid || save.isPending} className="w-full">
          {save.isPending ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
          {save.isPending ? t("common.saving") : editing ? t("common.save") : t("common.add")}
        </Button>
      </DialogContent>
    </Dialog>
  );
}

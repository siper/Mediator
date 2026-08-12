import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Quality, QualityProfile } from "@/lib/api/types";
import { MediaType, MEDIA_TYPE_SLUG, qualitiesForType, typeOptions } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { cn } from "@/lib/cn";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profile?: QualityProfile | null;
}

export function QualityProfileDialog({ open, onOpenChange, profile }: Props) {
  const { t } = useTranslation();
  const editing = Boolean(profile);
  const [name, setName] = useState("");
  const [type, setType] = useState<MediaType>(MediaType.Movie);
  const [allowed, setAllowed] = useState<string[]>([]);

  const availableQualities = qualitiesForType(type);

  useEffect(() => {
    if (!open) return;
    if (profile) {
      setType(profile.Type);
      setName(profile.Name);
      setAllowed(profile.Allowed.map((q) => q.Name));
    } else {
      setType(MediaType.Movie);
      setName("");
      setAllowed([]);
    }
  }, [open, profile]);

  const queryClient = useQueryClient();

  const toggle = (q: string) =>
    setAllowed((prev) => {
      const without = prev.filter((x) => x !== q);
      if (without.length === prev.length) {
        return [...prev, q].sort();
      }
      return without;
    });

  const saveBody = {
    Name: name,
    Type: Number(type),
    Allowed: availableQualities.filter((q) => allowed.includes(q.Name)),
    Cutoff: availableQualities.find((q) => q.Name === allowed[allowed.length - 1]) ?? availableQualities[0],
  };

  const save = useMutation({
    mutationFn: () => {
      if (profile) {
        return api.put<QualityProfile>(`/api/quality-profiles/${profile.Id}`, saveBody);
      }
      return api.post<QualityProfile>("/api/quality-profiles", saveBody);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["profiles"] });
      toast.success(editing ? t("messages.qualityProfileUpdated") : t("messages.qualityProfileCreated"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = name.trim() !== "" && allowed.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <>
          <DialogHeader>
            <DialogTitle>{editing ? t("profiles.titleEdit") : t("profiles.titleAdd")}</DialogTitle>
            <DialogDescription>
              {editing ? t("profiles.descriptionEdit") : t("profiles.descriptionAdd")}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">{t("profiles.name")}</label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t("profiles.namePlaceholder")} />
            </div>

            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">{t("profiles.mediaType")}</label>
              <Select value={String(type)} onValueChange={(v) => setType(Number(v) as MediaType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {typeOptions.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {t(`mediaTypeNav.${MEDIA_TYPE_SLUG[o.type]}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">{t("profiles.allowedQualities")}</label>
              <div className="flex flex-wrap gap-1.5">
                {availableQualities.map((q: Quality) => {
                  const selected = allowed.includes(q.Name);
                  return (
                    <button
                      key={q.Name}
                      type="button"
                      onClick={() => toggle(q.Name)}
                      className={cn(
                        "rounded-md border px-2.5 py-1 font-mono text-xs transition-colors",
                        selected
                          ? "border-transparent bg-primary text-primary-foreground"
                          : "border-input text-muted-foreground hover:text-foreground"
                      )}
                    >
                      {q.Name}
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">{t("profiles.cutoff")}</label>
              <div className="text-sm">{t("profiles.cutoffHelp")}</div>
              <Badge variant="secondary" className="font-mono text-xs">
                {allowed[allowed.length - 1] ?? t("common.dash")}
              </Badge>
            </div>
          </div>

          <Button
            onClick={() => save.mutate()}
            disabled={!valid || save.isPending}
            className="w-full"
          >
            {save.isPending ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
            {save.isPending ? t("common.saving") : editing ? t("common.save") : t("common.add")}
          </Button>
        </>
      </DialogContent>
    </Dialog>
  );
}

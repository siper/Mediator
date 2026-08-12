import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import { MEDIA_TYPE_SLUG, type Media } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface DeleteMediaButtonProps {
  media: Media;
}

export function DeleteMediaButton({ media }: DeleteMediaButtonProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [deleteFiles, setDeleteFiles] = useState(false);

  useEffect(() => {
    if (!open) setDeleteFiles(false);
  }, [open]);

  const remove = useMutation({
    mutationFn: () => {
      const qs = deleteFiles ? "?delete_files=true" : "";
      return api.del(`/api/media/${media.Id}${qs}`);
    },
    onSuccess: () => {
      setOpen(false);
      queryClient.invalidateQueries({ queryKey: ["media"] });
      queryClient.removeQueries({ queryKey: ["media", media.Id] });
      toast.success(t("messages.mediaDeleted", { name: media.Name }));
      navigate(`/${MEDIA_TYPE_SLUG[media.Type]}`);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  return (
    <>
      <Button variant="destructive" size="sm" onClick={() => setOpen(true)}>
        <Trash2 />
        {t("actions.delete")}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("deleteMedia.title")}</DialogTitle>
            <DialogDescription>
              {t("deleteMedia.description", { name: media.Name })}
            </DialogDescription>
          </DialogHeader>
          <label className="flex items-start gap-3 rounded-lg border border-border px-3 py-3 text-sm">
            <input
              type="checkbox"
              className="mt-0.5 size-4 accent-primary"
              checked={deleteFiles}
              onChange={(e) => setDeleteFiles(e.target.checked)}
            />
            <span>
              <span className="font-medium">{t("deleteMedia.deleteFiles")}</span>
              <span className="mt-0.5 block text-muted-foreground">
                {t("deleteMedia.deleteFilesHelp")}
              </span>
            </span>
          </label>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)} disabled={remove.isPending}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => remove.mutate()}
              disabled={remove.isPending}
            >
              {remove.isPending ? t("deleteMedia.deleting") : t("actions.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Library } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { MediaTypeBadge } from "@/components/media-type-badge";
import { LibraryDialog } from "@/components/library-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete } from "@/components/settings/shared";

export function LibrariesSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["libraries"],
    queryFn: () => api.get<Library[]>("/api/libraries"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Library | null>(null);
  const remove = useDelete("/api/libraries", "libraries");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (l: Library) => {
    setEditing(l);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.libraries"
        descriptionKey="settings.librariesDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("library.name")}</th>
              <th className="px-4 py-2 font-medium">{t("library.type")}</th>
              <th className="px-4 py-2 font-medium">{t("library.path")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((l) => (
              <tr key={l.Id}>
                <td className="px-4 py-2.5 font-medium">{l.Name}</td>
                <td className="px-4 py-2.5">
                  <MediaTypeBadge type={l.Type} />
                </td>
                <td className="px-4 py-2.5 font-mono text-xs">{l.Path}</td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(l)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(l.Id)} disabled={remove.isPending}>
                    <Trash2 className="size-4" />
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Button size="sm" onClick={openNew}>
        <Plus />
        {t("library.titleAdd")}
      </Button>

      <LibraryDialog open={dialogOpen} onOpenChange={setDialogOpen} library={editing} />
    </div>
  );
}

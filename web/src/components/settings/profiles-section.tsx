import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { QualityProfile } from "@/lib/api/types";
import { MEDIA_TYPE_SLUG } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { QualityProfileDialog } from "@/components/quality-profile-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete } from "@/components/settings/shared";

export function ProfilesSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["profiles"],
    queryFn: () => api.get<QualityProfile[]>("/api/quality-profiles"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<QualityProfile | null>(null);
  const remove = useDelete("/api/quality-profiles", "profiles");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (p: QualityProfile) => {
    setEditing(p);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.profiles"
        descriptionKey="settings.profilesDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("common.name")}</th>
              <th className="px-4 py-2 font-medium">{t("common.type")}</th>
              <th className="px-4 py-2 font-medium">{t("profiles.allowedQualities")}</th>
              <th className="px-4 py-2 font-medium">{t("profiles.cutoff")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((p) => (
              <tr key={p.Id}>
                <td className="px-4 py-2.5 font-medium">{p.Name}</td>
                <td className="px-4 py-2.5">
                  <Badge variant="secondary">
                    {t(`mediaTypeNav.${MEDIA_TYPE_SLUG[p.Type]}`)}
                  </Badge>
                </td>
                <td className="px-4 py-2.5">
                  <div className="flex flex-wrap gap-1">
                    {p.Allowed.map((q) => (
                      <Badge key={q.Name} variant="outline" className="font-mono text-[10px]">
                        {q.Name}
                      </Badge>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-2.5 font-mono text-xs">{p.Cutoff?.Name ?? t("common.dash")}</td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(p)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(p.Id)}>
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
        {t("profiles.titleAdd")}
      </Button>

      <QualityProfileDialog open={dialogOpen} onOpenChange={setDialogOpen} profile={editing} />
    </div>
  );
}

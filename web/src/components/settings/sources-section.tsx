import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Source } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { SourceDialog } from "@/components/source-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete, useToggleEnabled } from "@/components/settings/shared";

export function SourcesSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["sources"],
    queryFn: () => api.get<Source[]>("/api/sources"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Source | null>(null);
  const remove = useDelete("/api/sources", "sources");
  const toggle = useToggleEnabled("/api/sources", "sources");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (s: Source) => {
    setEditing(s);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.sources"
        descriptionKey="settings.sourcesDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("common.type")}</th>
              <th className="px-4 py-2 font-medium">{t("common.name")}</th>
              <th className="px-4 py-2 font-medium">{t("sources.settings")}</th>
              <th className="px-4 py-2 font-medium">{t("common.enabled")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((s) => (
              <tr key={s.Id}>
                <td className="px-4 py-2.5">
                  <Badge variant="secondary">{t(`sourceTypes.${s.Type}.label`, { defaultValue: s.Type })}</Badge>
                </td>
                <td className="px-4 py-2.5 font-medium">{s.Name}</td>
                <td className="max-w-sm truncate px-4 py-2.5 font-mono text-xs">
                  {s.Type === "tmdb"
                    ? (s.Settings?.api_key ? "api_key: ****" : t("sources.noApiKey"))
                    : t("common.dash")}
                </td>
                <td className="px-4 py-2.5">
                  <Switch
                    checked={s.Enabled}
                    onCheckedChange={(v) => toggle.mutate({ id: s.Id, enabled: v })}
                    disabled={toggle.isPending}
                    aria-label={s.Enabled ? t("sources.disable") : t("sources.enable")}
                  />
                </td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(s)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(s.Id)} disabled={remove.isPending}>
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
        {t("sources.titleAdd")}
      </Button>

      <SourceDialog open={dialogOpen} onOpenChange={setDialogOpen} source={editing} />
    </div>
  );
}

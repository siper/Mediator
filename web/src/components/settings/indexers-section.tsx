import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Indexer } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { IndexerDialog } from "@/components/indexer-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete, useToggleEnabled } from "@/components/settings/shared";

export function IndexersSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["indexers"],
    queryFn: () => api.get<Indexer[]>("/api/indexers"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Indexer | null>(null);
  const remove = useDelete("/api/indexers", "indexers");
  const toggle = useToggleEnabled("/api/indexers", "indexers");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (ix: Indexer) => {
    setEditing(ix);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.indexers"
        descriptionKey="settings.indexersDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("indexers.name")}</th>
              <th className="px-4 py-2 font-medium">{t("common.type")}</th>
              <th className="px-4 py-2 font-medium">{t("indexers.host")}</th>
              <th className="px-4 py-2 font-medium">{t("common.enabled")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((ix) => (
              <tr key={ix.Id}>
                <td className="px-4 py-2.5 font-medium">{ix.Name}</td>
                <td className="px-4 py-2.5">
                  <Badge variant="secondary">{ix.Type}</Badge>
                </td>
                <td className="max-w-sm truncate px-4 py-2.5 font-mono text-xs">
                  {ix.Settings?.endpoint ?? t("common.dash")}
                </td>
                <td className="px-4 py-2.5">
                  <Switch
                    checked={ix.Enabled}
                    onCheckedChange={(v) => toggle.mutate({ id: ix.Id, enabled: v })}
                    disabled={toggle.isPending}
                    aria-label={ix.Enabled ? t("indexers.disable") : t("indexers.enable")}
                  />
                </td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(ix)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(ix.Id)}>
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
        {t("indexers.titleAdd")}
      </Button>

      <IndexerDialog open={dialogOpen} onOpenChange={setDialogOpen} indexer={editing} />
    </div>
  );
}

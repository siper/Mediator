import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { DownloadClient } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { DownloadClientDialog } from "@/components/download-client-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete, useToggleEnabled } from "@/components/settings/shared";

export function ClientsSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["clients"],
    queryFn: () => api.get<DownloadClient[]>("/api/download-clients"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<DownloadClient | null>(null);
  const remove = useDelete("/api/download-clients", "clients");
  const toggle = useToggleEnabled("/api/download-clients", "clients");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (c: DownloadClient) => {
    setEditing(c);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.clients"
        descriptionKey="settings.clientsDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("clients.name")}</th>
              <th className="px-4 py-2 font-medium">{t("common.type")}</th>
              <th className="px-4 py-2 font-medium">{t("clients.host")}</th>
              <th className="px-4 py-2 font-medium">{t("common.enabled")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((c) => (
              <tr key={c.Id}>
                <td className="px-4 py-2.5 font-medium">{c.Name}</td>
                <td className="px-4 py-2.5">
                  <Badge variant="secondary">{c.Type}</Badge>
                </td>
                <td className="max-w-sm truncate px-4 py-2.5 font-mono text-xs">
                  {c.Type === "qbittorrent"
                    ? (c.Settings?.host ?? t("common.dash"))
                    : c.Type === "author_today"
                      ? (c.Settings?.token ? "****" : t("common.dash"))
                      : t("common.dash")}
                </td>
                <td className="px-4 py-2.5">
                  <Switch
                    checked={c.Enabled}
                    onCheckedChange={(v) => toggle.mutate({ id: c.Id, enabled: v })}
                    disabled={toggle.isPending}
                    aria-label={c.Enabled ? t("clients.disable") : t("clients.enable")}
                  />
                </td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(c)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(c.Id)}>
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
        {t("clients.titleAdd")}
      </Button>

      <DownloadClientDialog open={dialogOpen} onOpenChange={setDialogOpen} client={editing} />
    </div>
  );
}

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Proxy } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { ProxyDialog } from "@/components/proxy-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete, useToggleEnabled } from "@/components/settings/shared";

export function ProxiesSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["proxies"],
    queryFn: () => api.get<Proxy[]>("/api/proxies"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Proxy | null>(null);
  const remove = useDelete("/api/proxies", "proxies");
  const toggle = useToggleEnabled("/api/proxies", "proxies");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (p: Proxy) => {
    setEditing(p);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.proxies"
        descriptionKey="settings.proxiesDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("proxies.name")}</th>
              <th className="px-4 py-2 font-medium">{t("proxies.type")}</th>
              <th className="px-4 py-2 font-medium">{t("proxies.endpoint")}</th>
              <th className="px-4 py-2 font-medium">{t("common.enabled")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((p) => (
              <tr key={p.Id}>
                <td className="px-4 py-2.5 font-medium">{p.Name}</td>
                <td className="px-4 py-2.5">
                  <Badge variant="secondary">{t(`proxyTypes.${p.Type}.label`, { defaultValue: p.Type })}</Badge>
                </td>
                <td className="max-w-sm truncate px-4 py-2.5 font-mono text-xs">
                  {p.Endpoint}
                </td>
                <td className="px-4 py-2.5">
                  <Switch
                    checked={p.Enabled}
                    onCheckedChange={(v) => toggle.mutate({ id: p.Id, enabled: v })}
                    disabled={toggle.isPending}
                    aria-label={p.Enabled ? t("proxies.disable") : t("proxies.enable")}
                  />
                </td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(p)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(p.Id)} disabled={remove.isPending}>
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
        {t("proxies.titleAdd")}
      </Button>

      <ProxyDialog open={dialogOpen} onOpenChange={setDialogOpen} proxy={editing} />
    </div>
  );
}

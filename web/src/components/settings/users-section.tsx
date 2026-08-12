import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { User } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { UserDialog } from "@/components/user-dialog";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { useDelete } from "@/components/settings/shared";

export function UsersSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["users"],
    queryFn: () => api.get<User[]>("/api/users"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const remove = useDelete("/api/users", "users");

  const openNew = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (u: User) => {
    setEditing(u);
    setDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.users"
        descriptionKey="settings.usersDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("users.name")}</th>
              <th className="px-4 py-2 font-medium">{t("users.email")}</th>
              <th className="px-4 py-2 font-medium">{t("users.role")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((u) => (
              <tr key={u.Id}>
                <td className="px-4 py-2.5 font-medium">{u.Name}</td>
                <td className="px-4 py-2.5 text-muted-foreground">{u.Email}</td>
                <td className="px-4 py-2.5">
                  <Badge variant={u.Role === "admin" ? "default" : "secondary"}>
                    {u.Role === "admin" ? t("users.roleAdmin") : t("users.roleUser")}
                  </Badge>
                </td>
                <td className="px-4 py-2.5 text-right">
                  <Button size="icon" variant="ghost" onClick={() => openEdit(u)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button size="icon" variant="ghost" onClick={() => remove.mutate(u.Id)} disabled={remove.isPending}>
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
        {t("users.titleAdd")}
      </Button>

      <UserDialog open={dialogOpen} onOpenChange={setDialogOpen} user={editing} />
    </div>
  );
}

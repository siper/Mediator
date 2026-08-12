import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Setting } from "@/lib/api/types";
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
import { SettingsSectionHeader } from "@/components/settings-layout";

const KEY_ENABLED = "requests_enabled";
const KEY_AUTO_APPROVE = "auto_approve_requests";

function boolValue(settings: Setting[] | undefined, key: string, def: boolean): boolean {
  const s = settings?.find((x) => x.Key === key);
  if (!s) return def;
  return s.Value === "1" || s.Value === "true";
}

export function RequestsSection() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const settingsQuery = useQuery({
    queryKey: ["settings"],
    queryFn: () => api.get<Setting[]>("/api/settings"),
  });

  const update = useMutation({
    mutationFn: ({ key, value }: { key: string; value: string }) =>
      api.put<Setting>(`/api/settings/${key}`, { value }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings"] }),
    onError: (e: unknown) => toast.error((e as Error).message),
  });

  const toggle = (key: string, next: boolean) => {
    update.mutate({ key, value: next ? "true" : "false" });
    if (key === KEY_ENABLED) {
      queryClient.invalidateQueries({ queryKey: ["settings", "public"] });
    }
  };

  const enabled = boolValue(settingsQuery.data, KEY_ENABLED, true);
  const autoApprove = boolValue(settingsQuery.data, KEY_AUTO_APPROVE, false);

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.requests"
        descriptionKey="settings.requestsDescription"
      />

      {settingsQuery.isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner /> {t("common.loading")}
        </div>
      )}

      <div className="space-y-6 rounded-lg border border-border bg-card p-6">
        <ToggleRow
          label={t("settings.requestsEnable")}
          description={t("settings.requestsEnableDescription")}
          checked={enabled}
          onCheckedChange={(v) => toggle(KEY_ENABLED, v)}
          disabled={update.isPending}
        />
        <ToggleRow
          label={t("settings.requestsAutoApprove")}
          description={t("settings.requestsAutoApproveDescription")}
          checked={autoApprove}
          onCheckedChange={(v) => toggle(KEY_AUTO_APPROVE, v)}
          disabled={update.isPending}
        />
      </div>
    </div>
  );
}

function ToggleRow({
  label,
  description,
  checked,
  onCheckedChange,
  disabled,
}: {
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="space-y-1">
        <p className="text-sm font-medium leading-none">{label}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} disabled={disabled} />
    </div>
  );
}

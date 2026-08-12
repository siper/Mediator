import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { toast } from "sonner";
import { Play, Pencil } from "lucide-react";
import { api } from "@/lib/api/client";
import type { Task } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { SettingsSectionHeader } from "@/components/settings-layout";
import { TaskEditDialog } from "@/components/task-edit-dialog";

const STATUS_VARIANT: Record<string, "secondary" | "info" | "success" | "destructive"> = {
  idle: "secondary",
  running: "info",
  ok: "success",
  error: "destructive",
};

function formatTime(iso: string | null, t: TFunction): string {
  if (!iso) return t("common.dash");
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return t("common.dash");
  return d.toLocaleDateString();
}

function statusLabel(t: TFunction, status: string): string {
  return t(`badges.${status}`, { defaultValue: status });
}

export function TasksSection() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["tasks"],
    queryFn: () => api.get<Task[]>("/api/tasks"),
  });

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.tasks"
        descriptionKey="settings.tasksDescription"
      />

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-4 py-2 font-medium">{t("tasks.table.task")}</th>
              <th className="px-4 py-2 font-medium">{t("tasks.table.interval")}</th>
              <th className="px-4 py-2 font-medium">{t("tasks.table.enabled")}</th>
              <th className="px-4 py-2 font-medium">{t("tasks.table.lastRun")}</th>
              <th className="px-4 py-2 font-medium">{t("tasks.table.nextRun")}</th>
              <th className="px-4 py-2 font-medium">{t("tasks.table.status")}</th>
              <th className="px-4 py-2 text-right font-medium">{t("tasks.table.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {query.data?.map((task) => (
              <TaskRow key={task.Name} task={task} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function TaskRow({ task }: { task: Task }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);

  const run = useMutation({
    mutationFn: () => api.post(`/api/tasks/${task.Name}/run`),
    onSuccess: () => {
      toast.success(t("messages.taskQueued"));
      setTimeout(() => queryClient.invalidateQueries({ queryKey: ["tasks"] }), 1500);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  return (
    <>
      <TaskEditDialog open={editOpen} onOpenChange={setEditOpen} task={task} />
      <tr>
        <td className="px-4 py-2.5 font-medium">{t(`tasks.${task.Name}`, { defaultValue: task.Name })}</td>
        <td className="px-4 py-2.5 text-xs text-muted-foreground">{task.Interval}</td>
        <td className="px-4 py-2.5">
          <Badge variant={task.Enabled ? "default" : "secondary"}>{t(task.Enabled ? "common.yes" : "common.no")}</Badge>
        </td>
        <td className="px-4 py-2.5 text-xs text-muted-foreground">{formatTime(task.LastRun, t)}</td>
        <td className="px-4 py-2.5 text-xs text-muted-foreground">{formatTime(task.NextRun, t)}</td>
        <td className="px-4 py-2.5">
          <Badge variant={STATUS_VARIANT[task.LastStatus] ?? "secondary"}>{statusLabel(t, task.LastStatus)}</Badge>
          {task.LastError && (
            <span className="ml-2 text-xs text-destructive" title={task.LastError}>
              {t("tasks.error")}
            </span>
          )}
        </td>
        <td className="px-4 py-2.5 text-right">
          <Button size="sm" variant="ghost" onClick={() => run.mutate()} disabled={run.isPending}>
            <Play className="size-4" />
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setEditOpen(true)}>
            <Pencil className="size-4" />
          </Button>
        </td>
      </tr>
    </>
  );
}

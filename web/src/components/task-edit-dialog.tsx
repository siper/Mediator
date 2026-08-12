import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api/client";
import type { Task } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export function TaskEditDialog({ open, onOpenChange, task }: { open: boolean, onOpenChange: (open: boolean) => void, task: Task }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [interval, setInterval] = useState(task.Interval);
  const [enabled, setEnabled] = useState(task.Enabled);
  const [penalty, setPenalty] = useState(task.Settings?.penalty ?? "3");

  useEffect(() => {
    setInterval(task.Interval);
    setEnabled(task.Enabled);
    setPenalty(task.Settings?.penalty ?? "3");
  }, [task]);

  const save = useMutation({
    mutationFn: () => {
      const settings: Record<string, string> = { ...(task.Settings ?? {}) };
      if (task.Name === "cleanup-stuck") {
        settings.penalty = penalty;
      }
      return api.put<Task>(`/api/tasks/${task.Name}`, { interval, enabled, settings });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      toast.success(t("messages.taskUpdated"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("tasks.editTask")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("tasks.table.task")}</label>
            <div className="font-mono">{task.Name}</div>
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("tasks.interval")}</label>
            <Input value={interval} onChange={(e) => setInterval(e.target.value)} />
          </div>
          {task.Name === "cleanup-stuck" && (
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">{t("tasks.penalty")}</label>
              <Input
                type="number"
                min={1}
                value={penalty}
                onChange={(e) => setPenalty(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">{t("tasks.penaltyHint")}</p>
            </div>
          )}
          <div className="flex items-center justify-between">
            <label className="text-xs text-muted-foreground">{t("tasks.enabled")}</label>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
        </div>
        <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
            <Button onClick={() => save.mutate()} disabled={save.isPending}>{t("common.save")}</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

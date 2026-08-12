import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { DownloadClient } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

type ClientType = "qbittorrent" | "author_today";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  client?: DownloadClient | null;
}

export function DownloadClientDialog({ open, onOpenChange, client }: Props) {
  const { t } = useTranslation();
  const cfg = (type: ClientType) => {
    const label = t(`clientTypes.${type}.label`);
    const description = t(`clientTypes.${type}.description`);
    return { label, description };
  };
  const placeholder = (type: ClientType) =>
    type === "qbittorrent" ? t(`clientTypes.${type}.hostPlaceholder`) : undefined;

  const editing = Boolean(client);
  const [step, setStep] = useState<"pick" | "form">("pick");
  const [type, setType] = useState<ClientType>("qbittorrent");
  const [name, setName] = useState("");
  const [host, setHost] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState("");

  useEffect(() => {
    if (!open) return;
    if (client) {
      setType((client.Type as ClientType) ?? "qbittorrent");
      setName(client.Name);
      setHost(client.Settings?.host ?? "");
      setUsername(client.Settings?.username ?? "");
      setPassword(client.Settings?.password ?? "");
      setToken(client.Settings?.token ?? "");
      setStep("form");
    } else {
      setType("qbittorrent");
      setName("");
      setHost("");
      setUsername("");
      setPassword("");
      setToken("");
      setStep("pick");
    }
  }, [open, client]);

  const queryClient = useQueryClient();
  const c = cfg(type);

  const resolvedName = name.trim() || c.label;

  const save = useMutation({
    mutationFn: () => {
      const settings: Record<string, string> = {};
      if (type === "qbittorrent") {
        settings.host = host;
        settings.username = username;
        settings.password = password;
      } else if (type === "author_today") {
        settings.token = token;
      }
      const body = {
        Name: resolvedName,
        Type: type,
        Settings: settings,
        Enabled: client?.Enabled ?? true,
      };
      if (client) {
        return api.put(`/api/download-clients/${client.Id}`, body);
      }
      return api.post("/api/download-clients", body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["clients"] });
      toast.success(editing ? t("messages.clientUpdated") : t("messages.clientAdded"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const test = useMutation({
    mutationFn: () => {
      const settings: Record<string, string> = {};
      if (type === "qbittorrent") {
        settings.host = host;
        settings.username = username;
        settings.password = password;
      } else if (type === "author_today") {
        settings.token = token;
      }
      return api.post<{ status: string }>("/api/download-clients/test", {
        Name: resolvedName,
        Type: type,
        Settings: settings,
        Enabled: true,
      });
    },
    onSuccess: () => toast.success(t("messages.connectionSuccessful")),
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = type === "qbittorrent" ? host.trim() !== "" : token.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        {step === "pick" && !editing ? (
          <>
            <DialogHeader>
              <DialogTitle>{t("clients.titleAdd")}</DialogTitle>
              <DialogDescription>{t("clients.description")}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-2">
              {(["qbittorrent", "author_today"] as ClientType[]).map((tt) => {
                const cc = cfg(tt);
                return (
                  <button
                    key={tt}
                    onClick={() => {
                      setType(tt);
                      setStep("form");
                    }}
                    className="flex flex-col gap-1 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:border-primary/40 hover:bg-accent"
                  >
                    <span className="text-sm font-medium">{cc.label}</span>
                    <span className="text-xs text-muted-foreground">{cc.description}</span>
                  </button>
                );
              })}
            </div>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>{editing ? t("clients.titleEdit") : `Add ${c.label}`}</DialogTitle>
              <DialogDescription>{t("clients.connectionSettings")}</DialogDescription>
            </DialogHeader>

            {!editing && (
              <button
                onClick={() => setStep("pick")}
                className="inline-flex w-fit items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
              >
                <ArrowLeft className="size-3" />
                {t("clients.changeType")}
              </button>
            )}

            <div className="flex items-center gap-2">
              <Badge variant="secondary">{c.label}</Badge>
            </div>

            <div className="space-y-3">
              <div className="space-y-1">
                <label className="text-xs text-muted-foreground">{t("common.name")}</label>
                <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={c.label} />
              </div>
              {type === "qbittorrent" && (
                <>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t("clients.host")}</label>
                    <Input
                      value={host}
                      onChange={(e) => setHost(e.target.value)}
                      placeholder={placeholder(type)}
                      className="font-mono text-xs"
                    />
                  </div>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t("clients.username")}</label>
                    <Input value={username} onChange={(e) => setUsername(e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t("clients.password")}</label>
                    <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
                  </div>
                </>
              )}

              {type === "author_today" && (
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">{t("clients.accessToken")}</label>
                  <Input
                    type="password"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                    placeholder={t("clients.tokenPlaceholder")}
                    className="font-mono text-xs"
                  />
                </div>
              )}
            </div>

            <Button
              variant="outline"
              onClick={() => test.mutate()}
              disabled={test.isPending}
              className="w-full"
            >
              {test.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              {test.isPending ? t("clients.testing") : t("clients.testConnection")}
            </Button>

            <Button onClick={() => save.mutate()} disabled={!valid || save.isPending} className="w-full">
              {save.isPending ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
              {save.isPending ? t("common.saving") : editing ? t("common.save") : t("common.add")}
            </Button>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

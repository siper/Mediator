import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Proxy } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

type ProxyType = "http" | "socks5";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  proxy?: Proxy | null;
}

export function ProxyDialog({ open, onOpenChange, proxy }: Props) {
  const { t } = useTranslation();
  const cfg = (type: ProxyType) => ({
    label: t(`proxyTypes.${type}.label`),
    description: t(`proxyTypes.${type}.description`),
    placeholder: t(`proxyTypes.${type}.placeholder`),
  });

  const editing = Boolean(proxy);
  const [type, setType] = useState<ProxyType>("http");
  const [name, setName] = useState("");
  const [endpoint, setEndpoint] = useState("");

  useEffect(() => {
    if (!open) return;
    if (proxy) {
      setType((proxy.Type as ProxyType) ?? "http");
      setName(proxy.Name);
      setEndpoint(proxy.Endpoint);
    } else {
      setType("http");
      setName("");
      setEndpoint("");
    }
  }, [open, proxy]);

  const queryClient = useQueryClient();
  const c = cfg(type);

  const save = useMutation({
    mutationFn: () => {
      const body = {
        Name: name,
        Type: type,
        Endpoint: endpoint.trim(),
        Enabled: proxy?.Enabled ?? true,
      };
      if (proxy) {
        return api.put<Proxy>(`/api/proxies/${proxy.Id}`, body);
      }
      return api.post<Proxy>("/api/proxies", body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proxies"] });
      toast.success(editing ? t("messages.proxyUpdated") : t("messages.proxyAdded"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = name.trim() !== "" && endpoint.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{editing ? t("proxies.titleEdit") : t("proxies.titleAdd")}</DialogTitle>
          <DialogDescription>{t("proxies.description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("proxies.type")}</label>
            <Select value={type} onValueChange={(v) => setType(v as ProxyType)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(Object.keys({ http: 0, socks5: 0 }) as ProxyType[]).map((tt) => (
                  <SelectItem key={tt} value={tt}>
                    {cfg(tt).label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("common.name")}</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t("proxies.namePlaceholder")} />
          </div>

          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("proxies.endpoint")}</label>
            <Input
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder={c.placeholder}
              className="font-mono text-xs"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("proxies.description2")}</label>
            <p className="text-xs text-muted-foreground">{c.description}</p>
          </div>
        </div>

        <Button onClick={() => save.mutate()} disabled={!valid || save.isPending} className="w-full">
          {save.isPending ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
          {save.isPending ? t("common.saving") : editing ? t("common.save") : t("common.add")}
        </Button>
      </DialogContent>
    </Dialog>
  );
}

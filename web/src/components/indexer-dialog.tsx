import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Indexer } from "@/lib/api/types";
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

type IndexerType = "prowlarr" | "jackett";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  indexer?: Indexer | null;
}

export function IndexerDialog({ open, onOpenChange, indexer }: Props) {
  const { t } = useTranslation();
  const cfg = (type: IndexerType) => {
    const label = t(`indexerTypes.${type}.label`);
    const description = t(`indexerTypes.${type}.description`);
    const placeholder = t(`indexerTypes.${type}.hostPlaceholder`);
    return { label, description, placeholder };
  };

  const editing = Boolean(indexer);
  const [step, setStep] = useState<"pick" | "form">("pick");
  const [type, setType] = useState<IndexerType>("prowlarr");
  const [name, setName] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [apiKey, setApiKey] = useState("");

  useEffect(() => {
    if (!open) return;
    if (indexer) {
      setType((indexer.Type as IndexerType) ?? "prowlarr");
      setName(indexer.Name);
      setEndpoint(indexer.Settings?.endpoint ?? "");
      setApiKey(indexer.Settings?.api_key ?? "");
      setStep("form");
    } else {
      setType("prowlarr");
      setName("");
      setEndpoint("");
      setApiKey("");
      setStep("pick");
    }
  }, [open, indexer]);

  const queryClient = useQueryClient();
  const c = cfg(type);

  const resolvedName = name.trim() || c.label;

  const save = useMutation({
    mutationFn: () => {
      const body = {
        Name: resolvedName,
        Type: type,
        Settings: { endpoint, api_key: apiKey },
        Enabled: indexer?.Enabled ?? true,
      };
      if (indexer) {
        return api.put(`/api/indexers/${indexer.Id}`, body);
      }
      return api.post("/api/indexers", body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["indexers"] });
      toast.success(editing ? t("messages.indexerUpdated") : t("messages.indexerAdded"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const test = useMutation({
    mutationFn: () =>
      api.post<{ status: string }>("/api/indexers/test", {
        Name: resolvedName,
        Type: type,
        Settings: { endpoint, api_key: apiKey },
        Enabled: true,
      }),
    onSuccess: () => toast.success(t("messages.connectionSuccessful")),
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = endpoint.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        {step === "pick" && !editing ? (
          <>
            <DialogHeader>
              <DialogTitle>{t("indexers.titleAdd")}</DialogTitle>
              <DialogDescription>{t("indexers.description")}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-2">
              {(["prowlarr", "jackett"] as IndexerType[]).map((tt) => {
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
              <DialogTitle>{editing ? t("indexers.titleEdit") : `Add ${c.label}`}</DialogTitle>
              <DialogDescription>{t("indexers.endpointSettings")}</DialogDescription>
            </DialogHeader>

            {!editing && (
              <button
                onClick={() => setStep("pick")}
                className="inline-flex w-fit items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
              >
                <ArrowLeft className="size-3" />
                {t("indexers.changeType")}
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
              <div className="space-y-1">
                <label className="text-xs text-muted-foreground">{t("indexers.host")}</label>
                <Input
                  value={endpoint}
                  onChange={(e) => setEndpoint(e.target.value)}
                  placeholder={c.placeholder}
                  className="font-mono text-xs"
                />
                <p className="text-[11px] text-muted-foreground">{t("indexers.hostHelp")}</p>
              </div>
              <div className="space-y-1">
                <label className="text-xs text-muted-foreground">{t("indexers.apiKey")}</label>
                <Input value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
              </div>
            </div>

            <Button
              variant="outline"
              onClick={() => test.mutate()}
              disabled={!valid || test.isPending}
              className="w-full"
            >
              {test.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              {test.isPending ? t("indexers.testing") : t("indexers.testConnection")}
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

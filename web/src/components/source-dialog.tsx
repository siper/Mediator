import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Proxy, Source } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

type SourceType = "tmdb" | "author_today" | "musicbrainz" | "openlibrary";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  source?: Source | null;
}

export function SourceDialog({ open, onOpenChange, source }: Props) {
  const { t } = useTranslation();
  const cfg = (type: SourceType) => {
    const label = t(`sourceTypes.${type}.label`);
    const description = t(`sourceTypes.${type}.description`);
    return { label, description };
  };

  const editing = Boolean(source);
  const [step, setStep] = useState<"pick" | "form">("pick");
  const [type, setType] = useState<SourceType>("tmdb");
  const [name, setName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [language, setLanguage] = useState("en-US");
  const [proxyID, setProxyID] = useState<number | null>(null);

  const proxies = useQuery({
    queryKey: ["proxies"],
    queryFn: () => api.get<Proxy[]>("/api/proxies"),
  });

  useEffect(() => {
    if (!open) return;
    if (source) {
      setType((source.Type as SourceType) ?? "tmdb");
      setName(source.Name || source.Type);
      setApiKey(source.Settings?.api_key ?? "");
      setLanguage(source.Settings?.language ?? "en-US");
      setProxyID(source.ProxyID ?? null);
      setStep("form");
    } else {
      setType("tmdb");
      setName("");
      setApiKey("");
      setLanguage("en-US");
      setProxyID(null);
      setStep("pick");
    }
  }, [open, source]);

  const queryClient = useQueryClient();
  const c = cfg(type);
  const isTmdb = type === "tmdb";
  const resolvedName = name.trim() || c.label;

  const buildBody = () => {
    const settings: Record<string, string> = {};
    if (isTmdb) {
      settings.api_key = apiKey;
      settings.language = language;
    }
    return {
      Type: type,
      Name: resolvedName,
      Settings: settings,
      Enabled: source?.Enabled ?? true,
      ProxyID: proxyID,
    };
  };

  const save = useMutation({
    mutationFn: () => {
      const body = buildBody();
      if (source) {
        return api.put(`/api/sources/${source.Id}`, body);
      }
      return api.post("/api/sources", body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sources"] });
      queryClient.invalidateQueries({ queryKey: ["proxies"] });
      toast.success(editing ? t("messages.sourceUpdated") : t("messages.sourceAdded"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const test = useMutation({
    mutationFn: () => api.post<{ status: string }>("/api/sources/test", buildBody()),
    onSuccess: () => toast.success(t("messages.connectionSuccessful")),
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = !isTmdb || apiKey.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        {step === "pick" && !editing ? (
          <>
            <DialogHeader>
              <DialogTitle>{t("sources.titleAdd")}</DialogTitle>
              <DialogDescription>{t("sources.description")}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-2">
              {(["tmdb", "author_today", "musicbrainz", "openlibrary"] as SourceType[]).map((tt) => {
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
              <DialogTitle>{editing ? t("sources.titleEdit") : `Add ${c.label}`}</DialogTitle>
              <DialogDescription>
                {isTmdb ? t("sources.apiSettings") : t("sources.publicSettings")}
              </DialogDescription>
            </DialogHeader>

            {!editing && (
              <button
                onClick={() => setStep("pick")}
                className="inline-flex w-fit items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
              >
                <ArrowLeft className="size-3" />
                {t("sources.changeType")}
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
                <label className="text-xs text-muted-foreground">{t("sources.proxy")}</label>
                <Select
                  value={proxyID != null ? String(proxyID) : "__none__"}
                  onValueChange={(v) => setProxyID(v === "__none__" ? null : Number(v))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__none__">{t("sources.noProxy")}</SelectItem>
                    {proxies.data?.map((p) => (
                      <SelectItem key={p.Id} value={String(p.Id)}>
                        {p.Name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {isTmdb && (
                <>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t("sources.apiKey")}</label>
                    <Input
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder={t("sources.apiKeyPlaceholder")}
                      className="font-mono text-xs"
                    />
                  </div>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t("sources.language")}</label>
                    <Input
                      value={language}
                      onChange={(e) => setLanguage(e.target.value)}
                      placeholder="en-US"
                      className="font-mono text-xs"
                    />
                  </div>
                </>
              )}
            </div>

            <Button
              variant="outline"
              onClick={() => test.mutate()}
              disabled={test.isPending}
              className="w-full"
            >
              {test.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              {test.isPending ? t("sources.testing") : t("sources.testConnection")}
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

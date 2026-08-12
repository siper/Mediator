import { useState } from "react";
import { Link, Navigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, X, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { MediaRequest, PublicSettings, User } from "@/lib/api/types";
import { MEDIA_TYPE_SLUG } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CoverImg } from "@/components/ui/cover-img";
import { Spinner } from "@/components/ui/spinner";
import { MediaTypeBadge } from "@/components/media-type-badge";
import { useAuthStore } from "@/store/auth";

export function RequestsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.Role === "admin";

  const [rejectId, setRejectId] = useState<number | null>(null);
  const [rejectNotes, setRejectNotes] = useState("");

  const requestsQuery = useQuery({
    queryKey: ["requests"],
    queryFn: () => api.get<MediaRequest[]>("/api/requests"),
    refetchInterval: 15_000,
  });

  const publicQuery = useQuery({
    queryKey: ["settings", "public"],
    queryFn: () => api.get<PublicSettings>("/api/settings/public"),
  });
  if (publicQuery.data && publicQuery.data.requests_enabled !== "true") {
    return <Navigate to="/movies" replace />;
  }

  const usersQuery = useQuery({
    queryKey: ["users"],
    queryFn: () => api.get<User[]>("/api/users"),
    enabled: isAdmin && !!requestsQuery.data,
  });

  const userName = (id: number) => {
    if (id === user?.Id) return user?.Name ?? `#${id}`;
    return usersQuery.data?.find((u) => u.Id === id)?.Name ?? `#${id}`;
  };

  const approve = useMutation({
    mutationFn: (id: number) => api.post<MediaRequest>(`/api/requests/${id}/approve`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requests"] });
      toast.success(t("requests.approve"));
    },
    onError: (e: unknown) => toast.error((e as Error).message),
  });

  const reject = useMutation({
    mutationFn: ({ id, notes }: { id: number; notes: string }) =>
      api.post<MediaRequest>(`/api/requests/${id}/reject`, { notes }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requests"] });
      toast.success(t("requests.reject"));
      setRejectId(null);
      setRejectNotes("");
    },
    onError: (e: unknown) => toast.error((e as Error).message),
  });

  const cancel = useMutation({
    mutationFn: (id: number) => api.post<MediaRequest>(`/api/requests/${id}/cancel`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requests"] });
      toast.success(t("requests.cancel"));
    },
    onError: (e: unknown) => toast.error((e as Error).message),
  });

  return (
    <div className="mx-auto max-w-6xl px-8 py-8">
      <header className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("requests.title")}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("requests.description")}
        </p>
      </header>

      {requestsQuery.isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner /> {t("common.loading")}
        </div>
      )}
      {requestsQuery.isError && (
        <p className="text-sm text-destructive">
          {(requestsQuery.error as Error).message}
        </p>
      )}
      {!requestsQuery.isLoading &&
        (requestsQuery.data?.length ?? 0) === 0 && (
          <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
            <p className="text-sm text-muted-foreground">
              {t("requests.empty")}
            </p>
          </div>
        )}

      {requestsQuery.data && requestsQuery.data.length > 0 && (
        <>
          <div className="overflow-hidden rounded-lg border border-border bg-popover">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
                <tr>
                  <th className="px-4 py-2 font-medium">{t("requests.table.media")}</th>
                  <th className="px-4 py-2 font-medium">{t("requests.table.type")}</th>
                  <th className="px-4 py-2 font-medium">{t("requests.table.requester")}</th>
                  <th className="px-4 py-2 font-medium">{t("requests.table.status")}</th>
                  <th className="px-4 py-2 font-medium">{t("requests.table.created")}</th>
                  <th className="px-4 py-2 text-right font-medium">{t("requests.table.actions")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {requestsQuery.data.map((r) => {
                  const isPending = r.Status === "pending";
                  const isMine = r.UserId === user?.Id;
                  return (
                    <tr key={r.Id}>
                      <td className="max-w-sm px-4 py-2.5">
                        <div className="flex min-w-0 items-center gap-3">
                          <div className="h-14 w-10 shrink-0 overflow-hidden rounded-md border border-border bg-muted">
                            <CoverImg
                              src={r.Cover}
                              alt={r.Title}
                              className="h-full w-full object-cover"
                            />
                          </div>
                          <div className="min-w-0 flex-1">
                            {r.MediaID ? (
                              <Link
                                to={`/${MEDIA_TYPE_SLUG[r.Type]}/${r.MediaID}`}
                                className="block truncate font-medium text-primary hover:underline"
                                title={r.Title}
                              >
                                {r.Title}
                              </Link>
                            ) : (
                              <p className="block truncate font-medium" title={r.Title}>
                                {r.Title}
                              </p>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-2.5">
                        <MediaTypeBadge type={r.Type} />
                      </td>
                      <td className="px-4 py-2.5 text-muted-foreground">
                        {userName(r.UserId)}
                      </td>
                      <td className="px-4 py-2.5">
                        <RequestStatusBadge status={r.Status} />
                      </td>
                      <td className="px-4 py-2.5 text-muted-foreground">
                        {r.CreatedAt
                          ? new Date(r.CreatedAt).toLocaleDateString()
                          : ""}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        {isAdmin && isPending && (
                          <>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => approve.mutate(r.Id)}
                              disabled={approve.isPending}
                              title={t("requests.approve")}
                            >
                              <Check className="size-4" />
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => {
                                setRejectId(r.Id);
                                setRejectNotes("");
                              }}
                              disabled={reject.isPending}
                              title={t("requests.reject")}
                            >
                              <X className="size-4" />
                            </Button>
                          </>
                        )}
                        {isPending && isMine && (
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => cancel.mutate(r.Id)}
                            disabled={cancel.isPending}
                            title={t("requests.cancel")}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>

            {rejectId != null && (
              <RejectRow
                notes={rejectNotes}
                setNotes={setRejectNotes}
                onCancel={() => {
                  setRejectId(null);
                  setRejectNotes("");
                }}
                onSubmit={() => reject.mutate({ id: rejectId, notes: rejectNotes })}
                isPending={reject.isPending}
              />
            )}
          </div>
        </>
      )}

    </div>
  );
}

function RequestStatusBadge({ status }: { status: MediaRequest["Status"] }) {
  const { t } = useTranslation();
  let variant:
    | "default"
    | "success"
    | "warning"
    | "secondary"
    | "destructive" = "secondary";
  if (status === "approved") variant = "success";
  else if (status === "pending") variant = "warning";
  else if (status === "rejected" || status === "canceled") variant = "destructive";
  return (
    <Badge variant={variant} className="capitalize">
      {t(`requests.status.${status}`)}
    </Badge>
  );
}

function RejectRow({
  notes,
  setNotes,
  onCancel,
  onSubmit,
  isPending,
}: {
  notes: string;
  setNotes: (v: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
  isPending: boolean;
}) {
  const { t } = useTranslation();
  return (
    <tr className="bg-muted/30">
      <td colSpan={6} className="px-4 py-3">
        <div className="flex items-end gap-2">
          <input
            className="flex-1 rounded-md border border-input bg-popover px-3 py-1 text-sm outline-none placeholder:text-muted-foreground focus-within:ring-2 focus-within:ring-ring"
            placeholder={t("requests.rejectNotesPlaceholder")}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
          <Button size="sm" variant="ghost" onClick={onCancel} disabled={isPending}>
            {t("common.cancel")}
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={onSubmit}
            disabled={isPending}
          >
            {t("requests.rejecting")}
          </Button>
        </div>
      </td>
    </tr>
  );
}

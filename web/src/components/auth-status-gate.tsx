import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { authStatusQueryOptions } from "@/lib/auth-status";

export function AuthStatusGate({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation();
  const statusQuery = useQuery(authStatusQueryOptions);

  if (statusQuery.isPending) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (statusQuery.isError || statusQuery.data == null) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-background p-4">
        <p className="text-sm text-muted-foreground">{t("common.error")}</p>
        <Button variant="outline" onClick={() => statusQuery.refetch()}>
          {t("common.refresh")}
        </Button>
      </div>
    );
  }

  return <>{children}</>;
}

import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { authApi } from "@/lib/api/client";
import { createOidcState } from "@/lib/oidc-state";

export function OidcRedirect() {
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    const state = createOidcState();
    authApi
      .oidcLogin(state)
      .then(({ RedirectURL }) => {
        window.location.href = RedirectURL;
      })
      .catch((err) => {
        started.current = false;
        toast.error((err as Error).message);
      });
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <Loader2 className="size-6 animate-spin text-muted-foreground" />
    </div>
  );
}

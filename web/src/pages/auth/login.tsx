import { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { useQuery } from "@tanstack/react-query";
import { OidcRedirect } from "@/components/oidc-redirect";
import { LocalLoginForm } from "@/components/local-login-form";
import { authStatusQueryOptions, isOidcOnly } from "@/lib/auth-status";
import { createOidcState } from "@/lib/oidc-state";

export function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { login, loading } = useAuthStore();
  const statusQuery = useQuery(authStatusQueryOptions);

  const from = (location.state as { from?: string })?.from ?? "/movies";

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const error = params.get("error");
    if (error) {
      toast.error(params.get("error_description") ?? error);
      navigate("/login", { replace: true, state: location.state });
    }
  }, [location.search, location.state, navigate]);

  useEffect(() => {
    const { user, initialized } = useAuthStore.getState();
    if (initialized && user) {
      navigate(from, { replace: true });
    }
  }, [navigate, from]);

  if (statusQuery.isPending || !statusQuery.data) {
    return null;
  }

  if (isOidcOnly(statusQuery.data)) {
    return <OidcRedirect />;
  }

  const handleLocalLogin = async (email: string, password: string) => {
    try {
      await login(email, password);
      navigate(from, { replace: true });
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleOIDC = async () => {
    try {
      const state = createOidcState();
      const { RedirectURL } = await authApi.oidcLogin(state);
      window.location.href = RedirectURL;
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const loginEnabled = statusQuery.data.LoginEnabled;
  const registrationEnabled = statusQuery.data.RegistrationEnabled;
  const oidcEnabled = statusQuery.data.OIDCEnabled;

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-semibold tracking-tight">{t("auth.loginTitle")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("auth.loginDesc")}</p>
        </div>

        {loginEnabled ? (
          <LocalLoginForm loading={loading} onSubmit={handleLocalLogin} />
        ) : null}

        {oidcEnabled && (
          <>
            {loginEnabled && (
              <div className="relative pt-4">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t border-border" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-background px-2 text-muted-foreground">{t("auth.or")}</span>
                </div>
              </div>
            )}
            <Button variant="outline" onClick={handleOIDC} className="w-full">
              {t("auth.oidcLogin")}
            </Button>
          </>
        )}

        {registrationEnabled && (
          <div className="pt-4 text-center text-sm">
            <span className="text-muted-foreground">{t("auth.noAccount")} </span>
            <button
              onClick={() => navigate("/register")}
              className="font-medium text-foreground underline-offset-4 hover:underline"
            >
              {t("auth.register")}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

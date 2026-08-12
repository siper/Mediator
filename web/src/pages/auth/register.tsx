import { useEffect, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Loader2, UserPlus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { OidcRedirect } from "@/components/oidc-redirect";
import { authStatusQueryOptions, isOidcOnly } from "@/lib/auth-status";

export function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { register, loading } = useAuthStore();
  const statusQuery = useQuery(authStatusQueryOptions);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");

  useEffect(() => {
    const { user, initialized } = useAuthStore.getState();
    if (initialized && user) {
      navigate("/movies", { replace: true });
    }
  }, [navigate]);

  if (statusQuery.isPending || !statusQuery.data) {
    return null;
  }

  if (isOidcOnly(statusQuery.data)) {
    return <OidcRedirect />;
  }

  if (!statusQuery.data.RegistrationEnabled) {
    return <Navigate to="/login" replace />;
  }

  const valid = name.trim() && email.trim() && password.length >= 6 && password === confirm;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    try {
      await register(name, email, password);
      navigate("/movies", { replace: true });
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-semibold tracking-tight">{t("auth.registerTitle")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("auth.registerDesc")}</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("auth.name")}</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("auth.name")}
              autoComplete="name"
              autoFocus
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("auth.email")}</label>
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              autoComplete="email"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("auth.password")}</label>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="new-password"
            />
            {password && password.length < 6 && (
              <p className="text-xs text-destructive">{t("auth.passwordTooShort")}</p>
            )}
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("auth.confirmPassword")}</label>
            <Input
              type="password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              placeholder="••••••••"
              autoComplete="new-password"
            />
            {confirm && password !== confirm && (
              <p className="text-xs text-destructive">{t("auth.passwordMismatch")}</p>
            )}
          </div>
          <div className="pt-3">
            <Button type="submit" disabled={!valid || loading} className="w-full">
              {loading ? <Loader2 className="size-4 animate-spin" /> : <UserPlus className="size-4" />}
              {loading ? t("common.loading") : t("auth.register")}
            </Button>
          </div>
        </form>

        <div className="pt-4 text-center text-sm">
          <span className="text-muted-foreground">{t("auth.haveAccount")} </span>
          <button
            onClick={() => navigate("/login")}
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            {t("auth.signin")}
          </button>
        </div>
      </div>
    </div>
  );
}

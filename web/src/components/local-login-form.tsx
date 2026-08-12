import { useState } from "react";
import { Loader2, LogIn } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type LocalLoginFormProps = {
  loading: boolean;
  onSubmit: (email: string, password: string) => Promise<void>;
};

export function LocalLoginForm({ loading, onSubmit }: LocalLoginFormProps) {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !password.trim()) return;
    await onSubmit(email, password);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">{t("auth.email")}</label>
        <Input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          autoComplete="email"
          autoFocus
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">{t("auth.password")}</label>
        <Input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          autoComplete="current-password"
        />
      </div>
      <div className="pt-3">
        <Button type="submit" disabled={loading} className="w-full">
          {loading ? <Loader2 className="size-4 animate-spin" /> : <LogIn className="size-4" />}
          {loading ? t("common.loading") : t("auth.signin")}
        </Button>
      </div>
    </form>
  );
}

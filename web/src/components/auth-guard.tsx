import { useEffect } from "react";
import { Navigate, useLocation, Outlet } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@/store/auth";
import { OidcRedirect } from "@/components/oidc-redirect";
import { authStatusQueryOptions, isOidcOnly } from "@/lib/auth-status";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, initialized, checkAuth } = useAuthStore();
  const location = useLocation();
  const statusQuery = useQuery(authStatusQueryOptions);

  useEffect(() => {
    if (!initialized) {
      checkAuth();
    }
  }, [initialized, checkAuth]);

  if (!initialized || statusQuery.isPending || !statusQuery.data) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!user) {
    if (isOidcOnly(statusQuery.data)) {
      return <OidcRedirect />;
    }
    return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  }

  return <>{children}</>;
}

export function AdminGuard() {
  const { user } = useAuthStore();
  const statusQuery = useQuery(authStatusQueryOptions);

  if (!user) {
    if (statusQuery.isPending) {
      return (
        <div className="flex min-h-screen items-center justify-center bg-background">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      );
    }
    if (isOidcOnly(statusQuery.data)) {
      return <OidcRedirect />;
    }
    return <Navigate to="/login" replace />;
  }

  if (user.Role !== "admin") {
    return <Navigate to="/movies" replace />;
  }

  return <Outlet />;
}

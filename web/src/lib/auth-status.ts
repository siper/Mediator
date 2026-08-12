import { authApi } from "@/lib/api/client";
import type { AuthStatus } from "@/lib/api/types";

export const authStatusQueryKey = ["auth-status"] as const;

export const authStatusQueryOptions = {
  queryKey: authStatusQueryKey,
  queryFn: () => authApi.status(),
  staleTime: Infinity,
};

export function isOidcOnly(status: AuthStatus | undefined): boolean {
  return status != null && !status.LoginEnabled && status.OIDCEnabled;
}

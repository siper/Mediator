import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api/client";

export function useDelete(path: string, queryKey: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.del(`${path}/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [queryKey] }),
    onError: (e) => toast.error((e as Error).message),
  });
}

export function useToggleEnabled(path: string, queryKey: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      api.patch(`${path}/${id}/enabled`, { enabled }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [queryKey] }),
    onError: (e) => toast.error((e as Error).message),
  });
}

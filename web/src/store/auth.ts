import { create } from "zustand";
import { authApi, refreshSession } from "@/lib/api/client";
import type { User } from "@/lib/api/types";
import { useUIStore } from "@/store/ui";

interface AuthState {
  user: User | null;
  initialized: boolean;
  loading: boolean;
  checkAuth: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  setUser: (u: User | null) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  initialized: false,
  loading: false,
  checkAuth: async () => {
    try {
      const user = await authApi.me();
      set({ user, initialized: true });
      return;
    } catch {
      // me() may 401 when access token expired; refreshSession retries below
    }
    if (await refreshSession()) {
      try {
        const user = await authApi.me();
        set({ user, initialized: true });
        return;
      } catch {
        // fall through
      }
    }
    set({ user: null, initialized: true });
  },
  login: async (email, password) => {
    set({ loading: true });
    try {
      const res = await authApi.login(email, password);
      set({ user: res.User, loading: false });
    } catch (e) {
      set({ loading: false });
      throw e;
    }
  },
  register: async (name, email, password) => {
    set({ loading: true });
    try {
      const res = await authApi.register(name, email, password);
      set({ user: res.User, loading: false });
    } catch (e) {
      set({ loading: false });
      throw e;
    }
  },
  logout: async () => {
    await authApi.logout();
    useUIStore.getState().setLogoutDialogOpen(false);
    set({ user: null });
  },
  setUser: (u) => set({ user: u }),
}));

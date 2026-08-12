import { create } from "zustand";
import type { MediaType, SearchResult } from "@/lib/api/types";

interface UIState {
  searchOpen: boolean;
  addResult: SearchResult | null;
  searchType: MediaType | null;
  isLogoutDialogOpen: boolean;
  setSearchOpen: (open: boolean) => void;
  setAddResult: (r: SearchResult | null) => void;
  setSearchType: (t: MediaType | null) => void;
  setLogoutDialogOpen: (open: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  searchOpen: false,
  addResult: null,
  searchType: null,
  isLogoutDialogOpen: false,
  setSearchOpen: (open) => set({ searchOpen: open }),
  setAddResult: (r) => set({ addResult: r }),
  setSearchType: (t) => set({ searchType: t }),
  setLogoutDialogOpen: (open) => set({ isLogoutDialogOpen: open }),
}));

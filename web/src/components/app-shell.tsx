import { useMemo } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Film,
  Tv,
  BookOpen,
  Music,
  Heart,
  ListVideo,
  History as HistoryIcon,
  ClipboardList,
  Settings,
  Shield,
  Search,
  LogOut,
} from "lucide-react";
import { useUIStore } from "@/store/ui";
import { useAuthStore } from "@/store/auth";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Library, PublicSettings } from "@/lib/api/types";
import { MediaType } from "@/lib/api/types";
import { SearchOverlay } from "@/components/search-overlay";
import { AddMediaDialog } from "@/components/add-media-dialog";
import { LogoutConfirmDialog } from "@/components/logout-confirm-dialog";
import { MediatorLogo } from "@/components/mediator-logo";
import { cn } from "@/lib/cn";

interface NavItem {
  to: string;
  labelKey: string;
  icon: React.ComponentType<{ className?: string }>;
  disabled?: boolean;
  mediaType?: MediaType;
}

interface NavGroup {
  titleKey: string;
  items: NavItem[];
}

const NAV_GROUPS: NavGroup[] = [
  {
    titleKey: "nav.library",
    items: [
      { to: "/movies", labelKey: "nav.movies", icon: Film, mediaType: MediaType.Movie },
      { to: "/series", labelKey: "nav.series", icon: Tv, mediaType: MediaType.Series },
      { to: "/books", labelKey: "nav.books", icon: BookOpen, mediaType: MediaType.Book },
      { to: "/music", labelKey: "nav.music", icon: Music, mediaType: MediaType.MusicAlbum },
    ],
  },
      {
        titleKey: "nav.activity",
        items: [
          { to: "/wanted", labelKey: "nav.wanted", icon: Heart },
          { to: "/queue", labelKey: "nav.queue", icon: ListVideo },
          { to: "/history", labelKey: "nav.history", icon: HistoryIcon },
          { to: "/requests", labelKey: "nav.requests", icon: ClipboardList },
        ],
      },
];

function Sidebar() {
  const { t } = useTranslation();
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const setLogoutDialogOpen = useUIStore((s) => s.setLogoutDialogOpen);
  const { user } = useAuthStore();
  const isAdmin = user?.Role === "admin";

  const librariesQuery = useQuery({
    queryKey: ["libraries"],
    queryFn: () => api.get<Library[]>("/api/libraries"),
  });

  const publicSettingsQuery = useQuery({
    queryKey: ["settings", "public"],
    queryFn: () => api.get<PublicSettings>("/api/settings/public"),
  });
  const requestsEnabled = publicSettingsQuery.data?.requests_enabled === "true";

  const availableTypes = useMemo(() => {
    if (!librariesQuery.data) return null;
    return new Set(librariesQuery.data.map((l) => l.Type));
  }, [librariesQuery.data]);

  const visibleGroups = useMemo(() => {
    return NAV_GROUPS.map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        if (item.to === "/requests" && !requestsEnabled) return false;
        if (availableTypes && item.mediaType !== undefined && !availableTypes.has(item.mediaType))
          return false;
        return true;
      }),
    })).filter((group) => group.items.length > 0);
  }, [availableTypes, requestsEnabled]);

  const systemItems: NavItem[] = isAdmin
    ? [
        { to: "/admin/libraries", labelKey: "nav.administration", icon: Shield },
        { to: "/settings/general", labelKey: "nav.settings", icon: Settings },
      ]
    : [
        { to: "/settings/general", labelKey: "nav.settings", icon: Settings },
      ];

  return (
    <aside className="flex h-screen w-60 shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
      <div className="flex h-14 items-center gap-2 px-4">
        <MediatorLogo className="text-foreground" />
        <span className="font-semibold tracking-tight">Mediator</span>
      </div>

      <div className="px-3 pb-2">
        <button
          onClick={() => setSearchOpen(true)}
          className="flex w-full items-center gap-3 rounded-lg border border-sidebar-border bg-popover px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-sidebar-accent"
        >
          <Search className="size-4" />
          {t("common.search")}
        </button>
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-4">
        {visibleGroups.map((group) => (
          <div key={group.titleKey} className="space-y-1">
            <p className="px-3 pb-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              {t(group.titleKey)}
            </p>
            {group.items.map((item) => (
              <NavLink
                key={item.to}
                to={item.disabled ? "#" : item.to}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                    item.disabled && "pointer-events-none opacity-40",
                    isActive && !item.disabled
                      ? "bg-sidebar-accent text-foreground"
                      : "text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
                  )
                }
                onClick={(e) => item.disabled && e.preventDefault()}
              >
                <item.icon className="size-4" />
                {t(item.labelKey)}
              </NavLink>
            ))}
          </div>
        ))}
        <div className="space-y-1">
          <p className="px-3 pb-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            {t("nav.system")}
          </p>
          {systemItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-sidebar-accent text-foreground"
                    : "text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
                )
              }
            >
              <item.icon className="size-4" />
              {t(item.labelKey)}
            </NavLink>
          ))}
        </div>
      </nav>

      <div className="flex items-center justify-between border-t border-sidebar-border px-4 py-3 text-xs text-muted-foreground">
        <div className="flex flex-col gap-0.5">
          <span className="truncate font-medium text-foreground">{user?.Name ?? "—"}</span>
          <span>v0.1.0</span>
        </div>
        <button
          onClick={() => setLogoutDialogOpen(true)}
          className="rounded-md p-1.5 transition-colors hover:bg-sidebar-accent"
          aria-label={t("auth.logout")}
        >
          <LogOut className="size-4" />
        </button>
      </div>
    </aside>
  );
}

export function AppShell() {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
      <SearchOverlay />
      <AddMediaDialog />
      <LogoutConfirmDialog />
    </div>
  );
}

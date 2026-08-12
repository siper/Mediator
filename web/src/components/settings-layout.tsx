import { NavLink, Outlet } from "react-router-dom";
import {
  Library as LibraryIcon,
  Radar,
  Download,
  SlidersHorizontal,
  Database,
  CalendarClock,
  Network,
  Users,
  ClipboardList,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/cn";

interface SettingsNavItem {
  to: string;
  labelKey: string;
  icon: React.ComponentType<{ className?: string }>;
}

const ADMIN_NAV: SettingsNavItem[] = [
  { to: "/admin/libraries", labelKey: "settings.libraries", icon: LibraryIcon },
  { to: "/admin/indexers", labelKey: "settings.indexers", icon: Radar },
  { to: "/admin/clients", labelKey: "settings.clients", icon: Download },
  { to: "/admin/profiles", labelKey: "settings.profiles", icon: SlidersHorizontal },
  { to: "/admin/sources", labelKey: "settings.sources", icon: Database },
  { to: "/admin/proxies", labelKey: "settings.proxies", icon: Network },
  { to: "/admin/tasks", labelKey: "settings.tasks", icon: CalendarClock },
  { to: "/admin/requests", labelKey: "settings.requests", icon: ClipboardList },
  { to: "/admin/users", labelKey: "settings.users", icon: Users },
];

const USER_NAV: SettingsNavItem[] = [
  { to: "/settings/general", labelKey: "settings.nav.user", icon: SlidersHorizontal },
];

export function SettingsLayout({ admin = false }: { admin?: boolean }) {
  const { t } = useTranslation();
  const items = admin ? ADMIN_NAV : USER_NAV;
  const sectionKey = admin ? "settings.nav.admin" : "settings.nav.user";

  return (
    <div className="flex h-full">
      <aside className="flex w-60 shrink-0 flex-col border-r border-sidebar-border bg-sidebar">
        <div className="flex h-14 items-center px-4">
          <span className="text-sm font-semibold tracking-tight">{t("settings.title")}</span>
        </div>
        <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-3">
          <div className="space-y-1">
            <p className="px-3 pb-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              {t(sectionKey)}
            </p>
            {items.map((item) => (
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
      </aside>
      <div className="flex-1 overflow-y-auto">
        <Outlet />
      </div>
    </div>
  );
}

export function SettingsSectionHeader({
  titleKey,
  descriptionKey,
  descriptionArgs,
}: {
  titleKey: string;
  descriptionKey?: string;
  descriptionArgs?: Record<string, unknown>;
}) {
  const { t } = useTranslation();
  return (
    <div className="space-y-1">
      <h1 className="text-2xl font-semibold tracking-tight">{t(titleKey)}</h1>
      {descriptionKey ? <p className="text-sm text-muted-foreground">{t(descriptionKey, descriptionArgs)}</p> : null}
    </div>
  );
}

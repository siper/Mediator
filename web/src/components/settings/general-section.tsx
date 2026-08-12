import { useTranslation } from "react-i18next";
import { useTheme } from "@/components/theme-provider";
import { setLanguage } from "@/i18n";
import { SettingsSectionHeader } from "../settings-layout";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

const LANGUAGES = [
  { value: "en", labelKey: "settings.languageEn" },
  { value: "ru", labelKey: "settings.languageRu" },
] as const;

const THEMES = [
  { value: "light", labelKey: "settings.themeLight" },
  { value: "dark", labelKey: "settings.themeDark" },
  { value: "system", labelKey: "settings.themeSystem" },
] as const;

export function GeneralSection() {
  const { t, i18n } = useTranslation();
  const { theme, setTheme } = useTheme();

  const currentLang = i18n.language ? i18n.language.slice(0, 2) : "en";

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-8 py-8">
      <SettingsSectionHeader
        titleKey="settings.nav.user"
        descriptionKey="settings.userDescription"
      />

      <div className="space-y-6">
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none">{t("settings.theme")}</label>
          <p className="text-sm text-muted-foreground">{t("settings.themeDescription")}</p>
          <Select value={theme} onValueChange={(v) => setTheme(v as "light" | "dark" | "system")}>
            <SelectTrigger className="mt-1">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {THEMES.map((th) => (
                <SelectItem key={th.value} value={th.value}>
                  {t(th.labelKey)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium leading-none">{t("settings.language")}</label>
          <p className="text-sm text-muted-foreground">{t("settings.languageDescription")}</p>
          <Select value={currentLang} onValueChange={setLanguage}>
            <SelectTrigger className="mt-1">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {LANGUAGES.map((lng) => (
                <SelectItem key={lng.value} value={lng.value}>
                  {t(lng.labelKey)} ({lng.value.toUpperCase()})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>
  );
}

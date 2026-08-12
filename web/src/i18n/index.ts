import i18n from "i18next";
import { initReactI18next, I18nextProvider as I18nextProviderBase, useTranslation } from "react-i18next";
import React from "react";
import en from "./en";
import ru from "./ru";

const SUPPORTED = ["en", "ru"] as const;
const STORAGE_KEY = "media-lang";

function detectInitialLanguage(): string {
  if (typeof window === "undefined") return "en";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored && SUPPORTED.includes(stored as any)) return stored;
  const browser = navigator.language?.slice(0, 2);
  return browser === "ru" ? "ru" : "en";
}

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    ru: { translation: ru },
  },
  lng: detectInitialLanguage(),
  fallbackLng: "en",
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
  returnObjects: true,
});

export function setLanguage(lng: string) {
  localStorage.setItem(STORAGE_KEY, lng);
  i18n.changeLanguage(lng);
}

function I18nextProvider({ children }: { children: React.ReactNode }) {
  return React.createElement(I18nextProviderBase, { i18n, children });
}

export { I18nextProvider, useTranslation };
export default i18n;

export function I18nextTestProvider({ children }: { children: React.ReactNode }) {
  i18n.changeLanguage("en");
  return React.createElement(I18nextProviderBase, { i18n, children });
}

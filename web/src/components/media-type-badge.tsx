import { useTranslation } from "react-i18next";
import { MEDIA_TYPE_BADGE, MEDIA_TYPE_SLUG, type MediaType } from "@/lib/api/types";
import { cn } from "@/lib/cn";

interface MediaTypeBadgeProps {
  type: MediaType;
  className?: string;
}

export function MediaTypeBadge({ type, className }: MediaTypeBadgeProps) {
  const { t } = useTranslation();
  const slug = MEDIA_TYPE_SLUG[type];
  const label = slug ? t(`mediaTypeNav.${slug}`) : String(type);
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-sm border px-2 py-0.5 text-xs font-medium transition-colors",
        MEDIA_TYPE_BADGE[type],
        className
      )}
    >
      {label}
    </span>
  );
}

import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";

export function HistoryEventBadge({ event }: { event: string }) {
  const { t } = useTranslation();
  switch (event) {
    case "grabbed":
      return <Badge variant="info">{t("badges.grabbed")}</Badge>;
    case "imported":
      return <Badge variant="success">{t("badges.imported")}</Badge>;
    case "failed":
      return <Badge variant="destructive">{t("badges.failed")}</Badge>;
    default:
      return <Badge variant="secondary">{event}</Badge>;
  }
}

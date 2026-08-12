import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useUIStore } from "@/store/ui";
import { useAuthStore } from "@/store/auth";
import { useTranslation } from "react-i18next";

export function LogoutConfirmDialog() {
  const { t } = useTranslation();
  const { isLogoutDialogOpen, setLogoutDialogOpen } = useUIStore();
  const { logout } = useAuthStore();

  return (
    <Dialog open={isLogoutDialogOpen} onOpenChange={setLogoutDialogOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("auth.logout")}</DialogTitle>
          <DialogDescription>{t("auth.confirmLogout")}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setLogoutDialogOpen(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            onClick={async () => {
              setLogoutDialogOpen(false);
              await logout();
            }}
          >
            {t("auth.logout")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

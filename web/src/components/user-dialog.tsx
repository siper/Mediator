import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { User } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user?: User | null;
}

export function UserDialog({ open, onOpenChange, user }: Props) {
  const { t } = useTranslation();
  const editing = Boolean(user);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<string>("user");

  useEffect(() => {
    if (!open) return;
    if (user) {
      setName(user.Name);
      setEmail(user.Email);
      setRole(user.Role);
      setPassword("");
    } else {
      setName("");
      setEmail("");
      setRole("user");
      setPassword("");
    }
  }, [open, user]);

  const queryClient = useQueryClient();

  const save = useMutation({
    mutationFn: () => {
      const body: Record<string, unknown> = {
        Name: name,
        Email: email,
        Role: role,
      };
      if (password) {
        body.Password = password;
      }
      if (user) {
        return api.put(`/api/users/${user.Id}`, body);
      }
      return api.post("/api/users", { ...body, Password: password });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast.success(editing ? t("messages.userUpdated") : t("messages.userAdded"));
      onOpenChange(false);
    },
    onError: (e) => toast.error((e as Error).message),
  });

  const valid = name.trim() !== "" && email.trim() !== "" && (editing || password.length >= 6);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{editing ? t("users.titleEdit") : t("users.titleAdd")}</DialogTitle>
          <DialogDescription>{t("users.description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("users.name")}</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("users.email")}</label>
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">
              {t("users.password")}
              {editing && ` (${t("users.passwordKeepBlank")})`}
            </label>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">{t("users.role")}</label>
            <Select value={role} onValueChange={setRole}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">{t("users.roleUser")}</SelectItem>
                <SelectItem value="admin">{t("users.roleAdmin")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <Button onClick={() => save.mutate()} disabled={!valid || save.isPending} className="w-full">
          {save.isPending ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
          {save.isPending ? t("common.saving") : editing ? t("common.save") : t("common.add")}
        </Button>
      </DialogContent>
    </Dialog>
  );
}

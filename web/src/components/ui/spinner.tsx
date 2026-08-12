import { cn } from "@/lib/cn";

export function Spinner({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "size-4 animate-spin rounded-full border-2 border-muted border-t-foreground",
        className
      )}
    />
  );
}

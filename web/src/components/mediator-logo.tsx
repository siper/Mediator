import { cn } from "@/lib/cn";

export function MediatorLogo({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      className={cn("size-5 shrink-0", className)}
      aria-hidden
    >
      <path d="M3.5 19.5V4.5h3.2l5.3 9.7 5.3-9.7H20.5v15h-2.9V9.8L13.4 18h-2.8L6.4 9.8v9.7H3.5Z" />
    </svg>
  );
}

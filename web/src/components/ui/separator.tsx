import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/cn";

const separatorVariants = cva("shrink-0 bg-border", {
  variants: {
    orientation: {
      horizontal: "h-px w-full",
      vertical: "h-full w-px",
    },
  },
  defaultVariants: { orientation: "horizontal" },
});

export interface SeparatorProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof separatorVariants> {}

export function Separator({ className, orientation, ...props }: SeparatorProps) {
  return <div className={cn(separatorVariants({ orientation }), className)} {...props} />;
}

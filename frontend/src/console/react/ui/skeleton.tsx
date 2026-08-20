import { cn } from "@/console/react/lib/utils"

function Skeleton({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-[hsl(var(--muted))]", className)}
      {...props}
    />
  )
}

export { Skeleton }

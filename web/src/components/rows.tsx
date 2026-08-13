// rows.tsx — shared list row components (DocumentRow, SourceStatusBadge,
// EmptyState, SkeletonList) rendered with shadcn primitives.

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { FileText } from "lucide-react"
import { cn } from "@/lib/utils"

export function DocumentRow({
  title,
  relPath,
  sourceName,
  meta,
  onClick,
  onContext,
}: {
  title: string
  relPath?: string
  sourceName?: string
  meta?: React.ReactNode
  onClick?: () => void
  onContext?: (e: React.MouseEvent) => void
}) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onContextMenu={(e) => {
        e.preventDefault()
        onContext?.(e)
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter") onClick?.()
      }}
      className="flex items-center gap-3 rounded-md px-2 py-2 text-sm hover:bg-accent hover:text-accent-foreground cursor-pointer"
    >
      <FileText className="size-4 shrink-0 text-muted-foreground" />
      <span className="truncate font-medium">{title}</span>
      <span className="truncate text-xs text-muted-foreground">
        {[sourceName, relPath].filter(Boolean).join(" · ")}
      </span>
      {meta && <span className="ml-auto shrink-0 text-xs text-muted-foreground">{meta}</span>}
    </div>
  )
}

const STATUS_LABELS: Record<string, string> = {
  pending: "Pending",
  cloning: "Cloning",
  extracting: "Extracting",
  indexing: "Indexing",
  ready: "Ready",
  unavailable: "Unavailable",
  error: "Error",
}

export function SourceStatusBadge({ status, errorMessage }: { status?: string; errorMessage?: string }) {
  const s = status || "pending"
  const icon = s === "ready" ? "✓" : s === "error" ? "!" : s === "unavailable" ? "⌁" : "⋯"
  return (
    <Badge variant={s === "error" ? "destructive" : s === "ready" ? "secondary" : "outline"} title={errorMessage}>
      {icon} {STATUS_LABELS[s] || s}
    </Badge>
  )
}

export function SkeletonList({ rows = 6, height = 40 }: { rows?: number; height?: number }) {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} style={{ height }} className="w-full" />
      ))}
    </div>
  )
}

export function EmptyState({
  icon = "🗂",
  headline,
  body,
  action,
}: {
  icon?: string
  headline: string
  body?: string
  action?: { label: string; onClick: () => void }
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-12 text-center">
      <div className="text-3xl">{icon}</div>
      <h3 className="text-base font-semibold">{headline}</h3>
      {body && <p className="max-w-sm text-sm text-muted-foreground">{body}</p>}
      {action && (
        <Button onClick={action.onClick} className="mt-2">
          {action.label}
        </Button>
      )}
    </div>
  )
}

export function cnExport() {
  return cn
}

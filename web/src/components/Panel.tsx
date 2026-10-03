/**
 * File: web/src/components/Panel.tsx
 * Purpose: Dense bordered panel shell plus shared empty / unavailable /
 *          error states so every panel reports missing data the same way.
 */
import type { ReactNode } from "react"
import { ApiError } from "@/api/client"
import { cn } from "@/lib/utils"

export function Panel({
  title,
  actions,
  children,
  className,
  bodyClassName,
}: {
  title: string
  actions?: ReactNode
  children: ReactNode
  className?: string
  bodyClassName?: string
}) {
  return (
    <section className={cn("flex min-w-0 flex-col rounded-md border bg-card", className)} aria-label={title}>
      <header className="flex h-8 shrink-0 items-center justify-between gap-2 border-b px-3">
        <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">{title}</h2>
        {actions}
      </header>
      <div className={cn("min-h-0 flex-1 p-3", bodyClassName)}>{children}</div>
    </section>
  )
}

export function EmptyState({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("flex h-full min-h-16 items-center justify-center text-xs text-muted-foreground", className)}>
      {children}
    </div>
  )
}

export function Unavailable({ source, fields }: { source: string; fields: string[] }) {
  return (
    <EmptyState className="flex-col gap-1 text-center">
      <span>Not available</span>
      <span className="num text-[11px] text-muted-foreground/70">
        {source} missing {fields.join(", ")}
      </span>
    </EmptyState>
  )
}

export function QueryError({ error, what }: { error: unknown; what: string }) {
  const msg =
    error instanceof ApiError
      ? error.status === 0 || error.status >= 500
        ? `${what} unreachable`
        : error.status === 404
          ? `${what} not available (404)`
          : error.message
      : `${what} error`
  return <EmptyState className="text-bad/80">{msg}</EmptyState>
}

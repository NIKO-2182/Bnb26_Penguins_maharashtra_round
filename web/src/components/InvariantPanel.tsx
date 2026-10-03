/**
 * File: web/src/components/InvariantPanel.tsx
 * Purpose: Safety invariants. Oversell and duplicate-allocation counts,
 *          green at 0 and red otherwise.
 */
import { Panel, QueryError, Unavailable, EmptyState } from "@/components/Panel"
import { useMetrics } from "@/hooks/useMetrics"
import { cn, fmtInt } from "@/lib/utils"

function Check({ label, value }: { label: string; value: number | undefined }) {
  const known = value != null
  const ok = value === 0
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="flex items-center gap-2 text-xs">
        <span
          aria-hidden
          className={cn("size-2 rounded-full", !known ? "bg-muted-foreground/40" : ok ? "bg-ok" : "bg-bad")}
        />
        {label}
      </span>
      <span className={cn("num text-lg font-medium", !known ? "text-muted-foreground" : ok ? "text-ok" : "text-bad")}>
        {known ? fmtInt(value) : "—"}
        <span className="sr-only">{known ? (ok ? " (pass)" : " (fail)") : " (not reported)"}</span>
      </span>
    </div>
  )
}

export function InvariantPanel() {
  const { data, error, isLoading } = useMetrics()

  let content
  if (isLoading) content = <EmptyState>Loading</EmptyState>
  else if (error) content = <QueryError error={error} what="API" />
  else if (data && data.oversell_count == null && data.duplicate_count == null)
    content = <Unavailable source="/metrics" fields={["oversell_count", "duplicate_count"]} />
  else
    content = (
      <div className="divide-y">
        <Check label="Oversold seats" value={data?.oversell_count} />
        <Check label="Duplicate allocations" value={data?.duplicate_count} />
      </div>
    )

  return <Panel title="Invariants">{content}</Panel>
}

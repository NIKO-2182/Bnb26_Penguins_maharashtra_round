/**
 * File: web/src/components/LiveCounters.tsx
 * Purpose: Row of live counters from GET /metrics: RPS, 429s,
 *          seats left (left / total when total_seats exists), pool size.
 */
import type { ReactNode } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { TickNumber } from "@/components/TickNumber"
import { useMetrics } from "@/hooks/useMetrics"
import { fmtInt } from "@/lib/utils"

function Cell({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-1 px-4 py-3">
      <span className="text-[11px] uppercase tracking-wider text-muted-foreground">{label}</span>
      <div className="num truncate text-xl font-medium">{children}</div>
    </div>
  )
}

export function LiveCounters() {
  const { data, isLoading, isError } = useMetrics()

  const body = (v: number | undefined, suffix?: ReactNode) => {
    if (isLoading) return <Skeleton className="h-6 w-16" />
    if (isError || v == null) return <span className="text-muted-foreground">—</span>
    return (
      <>
        <TickNumber value={v} format={fmtInt} />
        {suffix}
      </>
    )
  }

  return (
    <section
      aria-label="Live counters"
      className="grid grid-cols-2 divide-x divide-y rounded-md border bg-card md:grid-cols-4 md:divide-y-0"
    >
      <Cell label="Requests / s">{body(data?.rps)}</Cell>
      <Cell label="429s">{body(data?.["429s"])}</Cell>
      <Cell label="Seats left">
        {body(
          data?.seats_left,
          data?.total_seats != null ? (
            <span className="text-sm text-muted-foreground"> / {fmtInt(data.total_seats)}</span>
          ) : null,
        )}
      </Cell>
      <Cell label="Pool size">{body(data?.pool_size)}</Cell>
    </section>
  )
}

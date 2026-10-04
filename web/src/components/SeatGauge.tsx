/**
 * File: web/src/components/SeatGauge.tsx
 * Purpose: Visual seat-pool allocation: how many seats remain, and how the
 *          allocated seats split between humans and bots.
 * Notes: Reads seats_bots / seats_humans from /metrics, derived by the API from
 *        the hash-chained ledger. Every field is optional so an older API
 *        degrades to a plain remaining-count instead of rendering nothing.
 */
import { Panel, QueryError, EmptyState } from "@/components/Panel"
import { Skeleton } from "@/components/ui/skeleton"
import { useMetrics } from "@/hooks/useMetrics"
import { fmtInt, fmtPct, cn } from "@/lib/utils"

export function SeatGauge() {
  const { data, isLoading, error } = useMetrics()

  if (isLoading) {
    return (
      <Panel title="Seat allocation">
        <Skeleton className="h-8 w-full" />
      </Panel>
    )
  }
  if (error) {
    return (
      <Panel title="Seat allocation">
        <QueryError error={error} what="API" />
      </Panel>
    )
  }

  const total = data?.total_seats ?? 500
  const left = data?.seats_left ?? 0
  const bots = data?.seats_bots ?? 0
  const humans = data?.seats_humans ?? 0
  const hasSplit = data?.seats_bots != null && data?.seats_humans != null

  const allocated = hasSplit ? Math.min(humans + bots, total) : Math.max(0, total - left)
  const unallocated = Math.max(0, total - allocated)

  // Segments are percentages of the whole pool, so they always sum to 100.
  const pct = (n: number) => (total > 0 ? (n / total) * 100 : 0)
  const humanPct = pct(humans)
  const botPct = pct(bots)
  const freePct = pct(unallocated)

  const botShare = allocated > 0 ? bots / allocated : 0

  return (
    <Panel
      title="Seat allocation"
      actions={
        <span className="num text-sm text-muted-foreground">
          {fmtInt(allocated)} / {fmtInt(total)} allocated
        </span>
      }
    >
      <div className="flex flex-col gap-3">
        <div
          className="flex h-8 w-full overflow-hidden rounded-md border bg-secondary/40"
          role="img"
          aria-label={`${humans} seats to humans, ${bots} to bots, ${unallocated} unallocated, of ${total}`}
        >
          <div
            className="flex items-center justify-center bg-brand/85 text-[11px] font-medium text-primary-foreground transition-[width] duration-500"
            style={{ width: `${humanPct}%` }}
          >
            {humanPct > 14 ? fmtInt(humans) : null}
          </div>
          <div
            className="flex items-center justify-center bg-bad/80 text-[11px] font-medium text-primary-foreground transition-[width] duration-500"
            style={{ width: `${botPct}%` }}
          >
            {botPct > 14 ? fmtInt(bots) : null}
          </div>
          <div style={{ width: `${freePct}%` }} />
        </div>

        <div className="flex flex-wrap items-center gap-x-5 gap-y-1.5 text-xs">
          <Key color="var(--brand)" label="Humans" value={fmtInt(humans)} />
          <Key color="var(--bad)" label="Bots" value={fmtInt(bots)} />
          <Key color="var(--muted-foreground)" label="Unallocated" value={fmtInt(unallocated)} />
          <span className="num ml-auto text-muted-foreground">remaining {fmtInt(left)}</span>
        </div>

        {!hasSplit ? (
          <EmptyState className="min-h-0">
            Human/bot split not reported by this API build
          </EmptyState>
        ) : allocated === 0 ? (
          <p className="text-xs text-muted-foreground">No seats allocated yet — run a simulation.</p>
        ) : (
          <p className="text-xs text-muted-foreground">
            Bots hold{" "}
            <span className={cn("num", botShare > 0.5 ? "text-bad" : "text-ok")}>{fmtPct(botShare)}</span> of
            allocated seats.
          </p>
        )}
      </div>
    </Panel>
  )
}

function Key({ color, label, value }: { color: string; label: string; value: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span aria-hidden className="size-2 rounded-sm" style={{ background: color }} />
      <span className="text-muted-foreground">{label}</span>
      <span className="num font-medium">{value}</span>
    </span>
  )
}
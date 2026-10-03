/**
 * File: web/src/components/HeroStat.tsx
 * Purpose: The largest element on the operator page: "Bots won X% of seats"
 *          from /metrics bot_share.
 */
import { Skeleton } from "@/components/ui/skeleton"
import { TickNumber } from "@/components/TickNumber"
import { QueryError } from "@/components/Panel"
import { useMetrics } from "@/hooks/useMetrics"
import { fmtInt } from "@/lib/utils"

export function HeroStat() {
  const { data, isLoading, error } = useMetrics()

  const allocated = data?.total_seats != null ? data.total_seats - data.seats_left : null
  const noSeatsYet = allocated === 0
  const hasValue = data?.bot_share != null && !noSeatsYet

  return (
    <section aria-label="Bot share of seats" className="flex flex-col justify-between gap-4 rounded-md border bg-card p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Bot share of seats</h2>
        {data?.mode && (
          <span className="num rounded-sm border px-1.5 py-0.5 text-[11px] uppercase text-muted-foreground">
            {data.mode}
          </span>
        )}
      </div>

      {isLoading ? (
        <Skeleton className="h-20 w-56" />
      ) : error ? (
        <QueryError error={error} what="API" />
      ) : hasValue ? (
        <div>
          <p className="text-sm text-muted-foreground">Bots won</p>
          <p className="flex items-baseline gap-3">
            <TickNumber
              value={(data.bot_share as number) * 100}
              format={(n) => `${n.toFixed(1)}%`}
              className="num text-7xl font-semibold leading-none tracking-tight md:text-8xl"
            />
            <span className="text-sm text-muted-foreground">of seats</span>
          </p>
        </div>
      ) : (
        <div>
          <p className="num text-7xl font-semibold leading-none text-muted-foreground/40 md:text-8xl">—</p>
          <p className="mt-2 text-sm text-muted-foreground">Waiting for traffic</p>
        </div>
      )}

      <p className="num text-xs text-muted-foreground">
        {allocated != null && data?.total_seats != null
          ? `${fmtInt(allocated)} of ${fmtInt(data.total_seats)} seats allocated`
          : "\u00a0"}
      </p>
    </section>
  )
}

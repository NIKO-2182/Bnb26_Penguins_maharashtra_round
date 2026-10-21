/**
 * File: web/src/components/OutcomeChart.tsx
 * Purpose: Bots vs humans seats won over time. The series is built client-side
 *          from each /metrics poll: seats won = total_seats - seats_left,
 *          split by bot_share. Cleared on admin reset.
 */
import { useEffect, useRef, useState } from "react"
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { Panel, QueryError, Unavailable, EmptyState } from "@/components/Panel"
import { useMetrics } from "@/hooks/useMetrics"
import { useAdminStore } from "@/store/adminStore"
import { fmtInt } from "@/lib/utils"
import { axisProps, gridProps, tooltipProps } from "@/lib/chart"

interface Point {
  t: number
  bots: number
  humans: number
}

const MAX_POINTS = 600

export function OutcomeChart() {
  const { data, dataUpdatedAt, error, isLoading } = useMetrics()
  const resetEpoch = useAdminStore((s) => s.resetEpoch)
  const [series, setSeries] = useState<Point[]>([])
  const startRef = useRef<number | null>(null)

  useEffect(() => {
    setSeries([])
    startRef.current = null
  }, [resetEpoch])

  useEffect(() => {
    if (!data || data.total_seats == null) return
    const won = Math.max(0, data.total_seats - data.seats_left)
    // Prefer the API's real per-winner split. Falling back to won*bot_share
    // invents the breakdown, which is the thing this chart is supposed to show.
    const bots = data.seats_bots != null ? data.seats_bots : Math.round(won * (data.bot_share ?? 0))
    const humans = data.seats_humans != null ? data.seats_humans : Math.max(0, won - bots)
    startRef.current ??= dataUpdatedAt
    const t = Math.max(0, Math.round((dataUpdatedAt - startRef.current) / 1000))
    setSeries((s) => {
      if (s.length === 0) {
        return [{ t: 0, bots, humans }]
      }
      const last = s[s.length - 1]
      if (last.t === t && last.bots === bots && last.humans === humans) return s
      if (last.bots === bots && last.humans === humans) {
        // Extend the current state to current time so the timeline advances smoothly
        if (s.length >= 2 && s[s.length - 2].bots === bots && s[s.length - 2].humans === humans) {
          return [...s.slice(0, -1), { t, bots, humans }]
        }
      }
      return [...s, { t, bots, humans }].slice(-MAX_POINTS)
    })
  }, [dataUpdatedAt, data])

  let content
  if (isLoading) content = <EmptyState>Loading</EmptyState>
  else if (error) content = <QueryError error={error} what="API" />
  else if (data && data.total_seats == null) content = <Unavailable source="/metrics" fields={["total_seats"]} />
  else if (series.length < 2) content = <EmptyState>Waiting for traffic</EmptyState>
  else
    content = (
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={series} margin={{ top: 4, right: 4, bottom: 0, left: -12 }}>
          <CartesianGrid {...gridProps} />
          <XAxis
            dataKey="t"
            type="number"
            domain={[0, (max: number) => Math.max(max, 10)]}
            {...axisProps}
            tickFormatter={(v) => `${Math.round(v)}s`}
            minTickGap={28}
          />
          <YAxis {...axisProps} tickFormatter={fmtInt} allowDecimals={false} />
          <Tooltip {...tooltipProps} labelFormatter={(v) => `t = ${Math.round(Number(v))}s`} formatter={(v) => fmtInt(Number(v))} />
          <Area
            type="stepAfter"
            dataKey="humans"
            name="Humans"
            stackId="1"
            stroke="var(--brand)"
            fill="var(--brand)"
            fillOpacity={0.25}
            isAnimationActive={false}
          />
          <Area
            type="stepAfter"
            dataKey="bots"
            name="Bots"
            stackId="1"
            stroke="var(--chart-2)"
            fill="var(--chart-2)"
            fillOpacity={0.2}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    )

  return (
    <Panel
      title="Seats won over time"
      actions={<Legend items={[["Humans", "var(--brand)"], ["Bots", "var(--chart-2)"]]} />}
      bodyClassName="h-56"
    >
      {content}
    </Panel>
  )
}

export function Legend({ items }: { items: [string, string][] }) {
  return (
    <ul className="flex items-center gap-3 text-[11px] text-muted-foreground">
      {items.map(([name, color]) => (
        <li key={name} className="flex items-center gap-1.5">
          <span aria-hidden className="size-2 rounded-[2px]" style={{ background: color }} />
          {name}
        </li>
      ))}
    </ul>
  )
}

/**
 * File: web/src/components/TrustHistogram.tsx
 * Purpose: Distribution of trust scores across the pool. Hidden entirely
 *          until /metrics exposes trust_buckets (counts over equal-width
 *          buckets spanning 0..1).
 */
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { Panel } from "@/components/Panel"
import { useMetrics } from "@/hooks/useMetrics"
import { axisProps, gridProps, tooltipProps } from "@/lib/chart"
import { fmtInt } from "@/lib/utils"

export function TrustHistogram() {
  const { data } = useMetrics()
  const buckets = data?.trust_buckets
  if (!buckets?.length) return null

  const n = buckets.length
  const rows = buckets.map((count, i) => ({
    range: `${(i / n).toFixed(2)}–${((i + 1) / n).toFixed(2)}`,
    label: (i / n).toFixed(1),
    count,
  }))

  return (
    <Panel title="Trust score distribution" bodyClassName="h-48">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={rows} margin={{ top: 4, right: 4, bottom: 0, left: -12 }} barCategoryGap={2}>
          <CartesianGrid {...gridProps} />
          <XAxis dataKey="label" {...axisProps} interval="preserveStartEnd" />
          <YAxis {...axisProps} tickFormatter={fmtInt} allowDecimals={false} />
          <Tooltip
            {...tooltipProps}
            cursor={{ fill: "var(--accent)" }}
            labelFormatter={(_, p) => `Trust ${p?.[0]?.payload?.range ?? ""}`}
            formatter={(v) => [fmtInt(Number(v)), "Users"]}
          />
          <Bar dataKey="count" fill="var(--brand)" fillOpacity={0.8} isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </Panel>
  )
}

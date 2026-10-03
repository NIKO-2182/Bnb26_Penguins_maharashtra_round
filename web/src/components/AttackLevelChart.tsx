/**
 * File: web/src/components/AttackLevelChart.tsx
 * Purpose: Bot share vs attack intensity, one line per mode, from the
 *          matrix runs in GET /results.
 */
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { Panel, QueryError, EmptyState } from "@/components/Panel"
import { Legend } from "@/components/OutcomeChart"
import { useResults } from "@/hooks/useMetrics"
import { byIntensity } from "@/lib/results"
import { axisProps, gridProps, tooltipProps } from "@/lib/chart"
import { fmtPct } from "@/lib/utils"

export function AttackLevelChart() {
  const { data, error, isLoading } = useResults()
  const rows = data ? byIntensity(data) : []

  let content
  if (isLoading) content = <EmptyState>Loading</EmptyState>
  else if (error) content = <QueryError error={error} what="Results" />
  else if (rows.length === 0) content = <EmptyState>No completed runs yet</EmptyState>
  else
    content = (
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={rows} margin={{ top: 4, right: 8, bottom: 0, left: -12 }}>
          <CartesianGrid {...gridProps} />
          <XAxis dataKey="intensity" {...axisProps} tickFormatter={(v) => `${v}×`} />
          <YAxis {...axisProps} tickFormatter={(v) => fmtPct(v, 0)} domain={[0, 1]} />
          <Tooltip
            {...tooltipProps}
            labelFormatter={(v) => `${v}× attack`}
            formatter={(v) => (v == null ? "—" : fmtPct(Number(v)))}
          />
          <Line
            type="monotone"
            dataKey="fcfs"
            name="FCFS"
            stroke="var(--chart-2)"
            strokeWidth={1.5}
            dot={{ r: 2.5, fill: "var(--chart-2)" }}
            connectNulls
            isAnimationActive={false}
          />
          <Line
            type="monotone"
            dataKey="fair"
            name="Fair"
            stroke="var(--brand)"
            strokeWidth={2}
            dot={{ r: 2.5, fill: "var(--brand)" }}
            connectNulls
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    )

  return (
    <Panel
      title="Bot share by attack level"
      actions={<Legend items={[["FCFS", "var(--chart-2)"], ["Fair", "var(--brand)"]]} />}
      bodyClassName="h-56"
    >
      {content}
    </Panel>
  )
}

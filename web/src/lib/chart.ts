/**
 * File: web/src/lib/chart.ts
 * Purpose: Shared Recharts styling so every chart reads the same:
 *          thin muted axes, faint grid, compact dark tooltip.
 */

export const axisProps = {
  stroke: "var(--muted-foreground)",
  tick: { fill: "var(--muted-foreground)", fontSize: 11, fontFamily: "var(--font-mono)" },
  tickLine: false,
  axisLine: { stroke: "var(--border)" },
} as const

export const gridProps = {
  stroke: "var(--border)",
  strokeDasharray: "2 4",
  vertical: false,
} as const

export const tooltipProps = {
  cursor: { stroke: "var(--border)" },
  contentStyle: {
    background: "var(--popover)",
    border: "1px solid var(--border)",
    borderRadius: 6,
    fontSize: 12,
    fontFamily: "var(--font-mono)",
    padding: "6px 8px",
  },
  labelStyle: { color: "var(--muted-foreground)", marginBottom: 2 },
  itemStyle: { color: "var(--foreground)", padding: 0 },
} as const

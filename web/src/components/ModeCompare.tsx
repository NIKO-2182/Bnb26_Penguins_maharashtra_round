/**
 * File: web/src/components/ModeCompare.tsx
 * Purpose: Side-by-side FCFS vs Fair using the latest run of each mode from
 *          GET /results.
 */
import type { ResultRun } from "@/api/client"
import { Panel, QueryError, EmptyState } from "@/components/Panel"
import { useResults } from "@/hooks/useMetrics"
import { latestPerMode } from "@/lib/results"
import { fmtInt, fmtPct } from "@/lib/utils"

const ROWS: { label: string; get: (r: ResultRun) => string; hero?: boolean }[] = [
  { label: "Bot share of seats", get: (r) => fmtPct(r.bot_share), hero: true },
  // The headline fairness number: 1.0 means a bot is no likelier to win than a
  // human. Below 1.0 is not automatically good -- it can mean humans are being
  // over-rejected -- so it is shown alongside the rejection rate.
  { label: "Bot advantage ratio", get: (r) => (r.bot_advantage_ratio == null ? "—" : r.bot_advantage_ratio.toFixed(3)), hero: true },
  { label: "Human win rate", get: (r) => fmtPct(r.human_win_rate) },
  { label: "Human false rejections", get: (r) => fmtPct(r.human_false_rejection_rate) },
  { label: "Humans denied", get: (r) => fmtInt(r.humans_denied) },
  { label: "Seats to bots", get: (r) => fmtInt(r.seats_bots) },
  { label: "Seats to humans", get: (r) => fmtInt(r.seats_humans) },
  { label: "Oversell / duplicates", get: (r) => `${fmtInt(r.oversell_count)} / ${fmtInt(r.duplicate_count)}` },
]

export function ModeCompare() {
  const { data, error, isLoading } = useResults()

  let content
  if (isLoading) content = <EmptyState>Loading</EmptyState>
  else if (error) content = <QueryError error={error} what="Results" />
  else if (!data?.length) content = <EmptyState>No completed runs yet</EmptyState>
  else {
    const { fcfs, fair } = latestPerMode(data)
    content = (
      <table className="w-full text-xs">
        <thead>
          <tr className="text-[11px] uppercase tracking-wider text-muted-foreground">
            <th scope="col" className="pb-2 text-left font-medium">
              <span className="sr-only">Metric</span>
            </th>
            <th scope="col" className="pb-2 text-right font-medium">FCFS</th>
            <th scope="col" className="pb-2 text-right font-medium text-brand">Fair</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {ROWS.map((row) => (
            <tr key={row.label}>
              <th scope="row" className="py-1.5 text-left font-normal text-muted-foreground">
                {row.label}
              </th>
              <td className={`num py-1.5 text-right ${row.hero ? "text-base font-medium" : ""}`}>
                {fcfs ? row.get(fcfs) : "—"}
              </td>
              <td className={`num py-1.5 text-right ${row.hero ? "text-base font-medium" : ""}`}>
                {fair ? row.get(fair) : "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    )
  }

  return <Panel title="FCFS vs Fair">{content}</Panel>
}

/**
 * File: web/src/components/HumanCost.tsx
 * Purpose: What Fair Drop costs real people: human false-rejection rate and
 *          human win rate vs the expected rate (seats / humans_in_pool).
 */
import { Panel, QueryError, Unavailable, EmptyState } from "@/components/Panel"
import { useMetrics } from "@/hooks/useMetrics"
import { fmtPct, fmtInt } from "@/lib/utils"

function Row({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-right">
        <span className="num text-lg font-medium">{value}</span>
        {sub && <span className="num ml-2 text-[11px] text-muted-foreground">{sub}</span>}
      </span>
    </div>
  )
}

export function HumanCost() {
  const { data, error, isLoading } = useMetrics()

  let content
  if (isLoading) content = <EmptyState>Loading</EmptyState>
  else if (error) content = <QueryError error={error} what="API" />
  else if (data) {
    const missing = (
      ["human_false_rejection_rate", "human_win_rate", "humans_in_pool", "total_seats"] as const
    ).filter((k) => data[k] == null)
    if (missing.length === 4) {
      content = <Unavailable source="/metrics" fields={missing} />
    } else {
      const expected =
        data.total_seats != null && data.humans_in_pool
          ? Math.min(1, data.total_seats / data.humans_in_pool)
          : null
      const ratio = expected && data.human_win_rate != null ? data.human_win_rate / expected : null
      content = (
        <div className="divide-y">
          <Row label="False rejections (humans)" value={fmtPct(data.human_false_rejection_rate)} />
          <Row
            label="Human win rate"
            value={fmtPct(data.human_win_rate)}
            sub={ratio != null ? `${ratio.toFixed(2)}× expected` : undefined}
          />
          <Row
            label="Expected (seats / humans in pool)"
            value={fmtPct(expected)}
            sub={
              data.humans_in_pool != null && data.total_seats != null
                ? `${fmtInt(data.total_seats)} / ${fmtInt(data.humans_in_pool)}`
                : undefined
            }
          />
          {missing.length > 0 && (
            <p className="num pt-2 text-[11px] text-muted-foreground/70">/metrics missing {missing.join(", ")}</p>
          )}
        </div>
      )
    }
  }

  return <Panel title="Human cost">{content}</Panel>
}

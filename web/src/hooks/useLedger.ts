/**
 * File: web/src/hooks/useLedger.ts
 * Purpose: Polls GET /ledger?limit=N every second and returns events
 *          newest-first with a stable per-event key for row animation.
 */
import { useQuery } from "@tanstack/react-query"
import { getLedger, type LedgerEvent } from "@/api/client"

export interface KeyedEvent extends LedgerEvent {
  key: string
}

function toMillis(ts: string | number): number {
  if (typeof ts === "number") return ts < 1e12 ? ts * 1000 : ts
  const n = Date.parse(ts)
  return Number.isNaN(n) ? 0 : n
}

function keyEvents(events: LedgerEvent[]): KeyedEvent[] {
  const seen = new Map<string, number>()
  return [...events]
    .sort((a, b) => toMillis(b.timestamp) - toMillis(a.timestamp))
    .map((e) => {
      const base = `${e.timestamp}|${e.user_id}|${e.event_type}|${e.reason_code ?? ""}`
      const n = seen.get(base) ?? 0
      seen.set(base, n + 1)
      return { ...e, key: `${base}|${n}` }
    })
}

export function useLedger(limit = 50) {
  return useQuery({
    queryKey: ["ledger", limit],
    queryFn: () => getLedger(limit),
    select: keyEvents,
    refetchInterval: 1000,
    retry: false,
  })
}

/**
 * File: web/src/lib/results.ts
 * Purpose: Helpers over GET /results run records: latest run per mode and
 *          latest run per (mode, intensity).
 */
import type { Mode, ResultRun } from "@/api/client"

function order(run: ResultRun, index: number): number {
  const t = run.finished_at ? Date.parse(run.finished_at) : NaN
  return Number.isNaN(t) ? index : t
}

function latestBy(runs: ResultRun[], keyOf: (r: ResultRun) => string): Map<string, ResultRun> {
  const best = new Map<string, { run: ResultRun; o: number }>()
  runs.forEach((run, i) => {
    const k = keyOf(run)
    const o = order(run, i)
    const cur = best.get(k)
    if (!cur || o >= cur.o) best.set(k, { run, o })
  })
  return new Map([...best].map(([k, v]) => [k, v.run]))
}

export function latestPerMode(runs: ResultRun[]): Partial<Record<Mode, ResultRun>> {
  const m = latestBy(runs, (r) => r.mode)
  return { fcfs: m.get("fcfs"), fair: m.get("fair") }
}

export function byIntensity(runs: ResultRun[]) {
  const m = latestBy(runs, (r) => `${r.mode}|${r.intensity}`)
  const intensities = [...new Set(runs.map((r) => r.intensity))].sort((a, b) => a - b)
  return intensities.map((x) => ({
    intensity: x,
    fcfs: m.get(`fcfs|${x}`)?.bot_share ?? null,
    fair: m.get(`fair|${x}`)?.bot_share ?? null,
  }))
}

/**
 * File: web/src/lib/results.ts
 * Purpose: Helpers over GET /results run records: latest run per mode and
 *          latest run per (mode, intensity).
 */
import type { Mode, ResultRun } from "@/api/client"

/** Defensive: never let a malformed payload throw inside a render. */
function safeRuns(input: unknown): ResultRun[] {
  if (!Array.isArray(input)) return []
  return input.filter((r): r is ResultRun => !!r && typeof r === "object")
}

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
  const m = latestBy(safeRuns(runs), (r) => r.mode)
  return { fcfs: m.get("fcfs"), fair: m.get("fair") }
}

export function byIntensity(runs: ResultRun[]) {
  const safe = safeRuns(runs)
  const m = latestBy(safe, (r) => `${r.mode}|${r.intensity}`)
  // The API does not report an attack intensity per run, so fall back to a
  // single bucket keyed on the mode instead of producing an empty chart.
  const known = safe.filter((r) => Number.isFinite(r.intensity))
  const intensities = known.length
    ? [...new Set(known.map((r) => r.intensity))].sort((a, b) => a - b)
    : [0]
  return intensities.map((x) => ({
    intensity: x,
    fcfs: m.get(`fcfs|${x}`)?.bot_share ?? latestFor(safe, "fcfs")?.bot_share ?? null,
    fair: m.get(`fair|${x}`)?.bot_share ?? latestFor(safe, "fair")?.bot_share ?? null,
  }))
}

/** Most recent run of a given mode, ignoring intensity. */
function latestFor(safe: ResultRun[], mode: Mode): ResultRun | undefined {
  return latestBy(safe, (r) => r.mode).get(mode)
}

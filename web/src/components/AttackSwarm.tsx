/**
 * File: web/src/components/AttackSwarm.tsx
 * Purpose: Live visual of clients arriving at the drop. Every particle is a real
 *          ledger event, coloured by who it was and how it ended.
 * Notes: This is deliberately NOT decorative. Points come from GET /ledger,
 *          which is the append-only hash-chained log, so the animation cannot
 *          claim activity the system did not actually observe. Motion is
 *          suppressed under prefers-reduced-motion.
 */
import { useEffect, useMemo, useRef, useState } from "react"
import { Panel } from "@/components/Panel"
import { useQuery } from "@tanstack/react-query"
import { getDecidedLedger, type LedgerEvent } from "@/api/client"
import { keyEvents, type KeyedEvent } from "@/hooks/useLedger"
import { useAdminStore } from "@/store/adminStore"
import { cn } from "@/lib/utils"

const MAX_POINTS = 260

/** Distinct bot profiles, so the swarm shows attack variety not just volume. */
const BOT_PROFILES = new Set(["bot_naive", "bot_solver", "bot_distributed", "bot_retry"])

function profileOf(userId: string): string {
  const m = /^([a-z_]+)_/.exec(userId)
  return m ? m[1] : "unknown"
}

/** Why an arrival ended the way it did, mapped to a tone. */
function toneOf(reason: string | null): "win" | "block" | "late" | "idle" {
  if (reason === "seat_granted") return "win"
  if (!reason) return "idle"
  if (reason === "sold_out" || reason === "not_selected_draw") return "late"
  if (reason.startsWith("rejected") || reason === "duplicate_claim") return "block"
  return "idle"
}

const TONE_STYLE: Record<string, { dot: string; label: string }> = {
  win: { dot: "bg-ok", label: "won a seat" },
  block: { dot: "bg-bad", label: "blocked" },
  late: { dot: "bg-muted-foreground/40", label: "arrived after close" },
  idle: { dot: "bg-warn", label: "other" },
}

interface Particle {
  key: string
  x: number
  y: number
  tone: string
  bot: boolean
}

export function AttackSwarm() {
  // The server filters out the "sold_out" tail, which is ~94% of the ledger
  // and would otherwise fill any newest-N window. See /ledger?decided=1.
  const { data } = useQuery({
    queryKey: ["ledger-decided"],
    queryFn: () => getDecidedLedger(500),
    select: (rows: LedgerEvent[]) => keyEvents(rows) as KeyedEvent[],
    refetchInterval: 1000,
    retry: false,
  })
  const resetEpoch = useAdminStore((s) => s.resetEpoch)
  const [burst, setBurst] = useState(0)
  const seenRef = useRef<Set<string>>(new Set())
  const reduced = usePrefersReducedMotion()

  useEffect(() => {
    seenRef.current = new Set()
  }, [resetEpoch])

  // New arrivals since the previous poll drive the "burst" flourish only.
  useEffect(() => {
    if (!data?.length) return
    let fresh = 0
    for (const e of data) {
      if (!seenRef.current.has(e.key)) {
        seenRef.current.add(e.key)
        fresh++
      }
      if (seenRef.current.size > 4000) break
    }
    if (fresh > 0 && !reduced) setBurst((b) => Math.min(b + fresh, 400))
  }, [data, reduced])

  useEffect(() => {
    if (reduced || burst === 0) return
    const t = setTimeout(() => setBurst(0), 900)
    return () => clearTimeout(t)
  }, [burst, reduced])

  // Prefer events that reached a real decision, then fall back to the tail.
  // During a flash sale ~95% of all traffic is "sold_out" arriving after the
  // pool closed, so taking the newest N events yields a wall of identical grey
  // dots and hides the opening rush entirely -- which is the only part of the
  // run worth looking at.
  const particles = useMemo<Particle[]>(() => {
    if (!data?.length) return []

    const out: Particle[] = []
    for (const e of data.slice(0, MAX_POINTS)) {
      let h = 2166136261
      for (let i = 0; i < e.key.length; i++) {
        h ^= e.key.charCodeAt(i)
        h = Math.imul(h, 16777619)
      }
      const a = Math.abs(h)
      const b = Math.abs(Math.imul(h ^ 0x9e3779b9, 2246822519))
      const profile = profileOf(e.user_id)
      out.push({
        key: e.key,
        x: 4 + (a % 92),
        y: 6 + (b % 88),
        tone: toneOf(e.reason_code),
        bot: BOT_PROFILES.has(profile) || e.user_type === "bot",
      })
    }
    return out
  }, [data])

  const live = particles.length
  const bots = particles.filter((p) => p.bot).length
  const humans = live - bots

  return (
    <Panel
      title="Live arrivals"
      actions={
        <span className="num text-sm text-muted-foreground">
          {live} in window
          {!reduced && burst > 0 ? <span className="ml-2 text-ok">+{burst} new</span> : null}
        </span>
      }
    >
      <div className="flex flex-col gap-3">
        <div
          className="relative h-40 w-full overflow-hidden rounded-md border bg-secondary/25"
          role="img"
          aria-label={`${live} recent arrivals: ${bots} bots, ${humans} humans`}
        >
          {/* Baseline: the 500-seat pool as a faint rail */}
          <div className="pointer-events-none absolute inset-x-0 bottom-2 flex items-center gap-2 px-3">
            <span className="h-px flex-1 bg-border" />
            <span className="text-[10px] uppercase tracking-wider text-muted-foreground/60">
              500 seats
            </span>
            <span className="h-px flex-1 bg-border" />
          </div>

          {particles.map((p) => (
            <span
              key={p.key}
              className={cn(
                "absolute size-1.5 rounded-full transition-opacity duration-500",
                TONE_STYLE[p.tone]?.dot ?? TONE_STYLE.idle.dot,
                p.bot ? "opacity-95" : "opacity-60",
                !reduced && burst > 0 && "animate-pulse",
              )}
              style={{ left: `${p.x}%`, top: `${p.y}%` }}
              title={`${p.bot ? "bot" : "human"} — ${TONE_STYLE[p.tone]?.label ?? p.tone}`}
            />
          ))}

          {live === 0 && (
            <div className="absolute inset-0 flex items-center justify-center text-xs text-muted-foreground">
              No arrivals yet — run a simulation
            </div>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs">
          <Stat label="Bots" value={bots} dot="bg-bad" />
          <Stat label="Humans" value={humans} dot="bg-brand" />
          <div className="ml-auto flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
            {Object.entries(TONE_STYLE).map(([k, v]) => (
              <span key={k} className="flex items-center gap-1">
                <span aria-hidden className={cn("size-1.5 rounded-full", v.dot)} />
                {v.label}
              </span>
            ))}
          </div>
        </div>
      </div>
    </Panel>
  )
}

function Stat({ label, value, dot }: { label: string; value: number; dot: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span aria-hidden className={cn("size-2 rounded-sm", dot)} />
      <span className="text-muted-foreground">{label}</span>
      <span className="num font-medium">{value}</span>
    </span>
  )
}

function usePrefersReducedMotion(): boolean {
  const [reduced, setReduced] = useState(false)
  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)")
    setReduced(mq.matches)
    const on = () => setReduced(mq.matches)
    mq.addEventListener?.("change", on)
    return () => mq.removeEventListener?.("change", on)
  }, [])
  return reduced
}
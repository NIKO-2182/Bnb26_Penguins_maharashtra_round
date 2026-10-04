/**
 * File: web/src/api/client.ts
 * Purpose: Typed fetch wrappers for the Fair Drop API (docs/API_CONTRACT.md).
 *          All calls go through /api, which nginx (and the Vite dev server)
 *          proxies to api:8080 with the prefix stripped.
 * Notes:   Fields marked optional are "needed" in FRONTEND_SPEC §7 and may not
 *          exist yet. Panels check for them and hide themselves when absent.
 */

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api"

export class ApiError extends Error {
  status: number
  body: unknown
  constructor(status: number, message: string, body: unknown) {
    super(message)
    this.status = status
    this.body = body
  }
}

function errorMessage(body: unknown, fallback: string): string {
  if (body && typeof body === "object") {
    const b = body as Record<string, unknown>
    for (const key of ["error", "message", "detail", "reason_code"]) {
      if (typeof b[key] === "string") return b[key] as string
    }
  }
  if (typeof body === "string" && body.trim()) return body.trim()
  return fallback
}

export async function request<T>(base: string, path: string, init: RequestInit = {}): Promise<T> {
  let res: Response
  try {
    res = await fetch(`${base}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...init.headers,
      },
    })
  } catch {
    throw new ApiError(0, "Service unreachable", null)
  }
  const text = await res.text()
  let body: unknown = text
  try {
    body = text ? JSON.parse(text) : null
  } catch {
    /* non-JSON body, keep raw text */
  }
  if (!res.ok) {
    const fallback = res.status === 502 || res.status === 504 ? "Service unreachable" : `HTTP ${res.status}`
    throw new ApiError(res.status, errorMessage(body, fallback), body)
  }
  return body as T
}

const api = <T>(path: string, init?: RequestInit) => request<T>(API_BASE, path, init)
const authHeader = (token: string) => ({ Authorization: `Bearer ${token}` })

export type Mode = "fair" | "fcfs"
export type UserType = "human" | "bot"

export interface Metrics {
  rps: number
  "429s": number
  seats_left: number
  pool_size: number
  bot_share: number | null
  tp?: number
  fp?: number
  fn?: number
  tn?: number
  precision?: number
  recall?: number
  human_false_rejection_rate?: number
  by_reason?: Record<string, number>
  by_profile?: Record<string, number>
  total_seats?: number
  human_win_rate?: number
  humans_in_pool?: number
  /** Seats actually granted, split by winner. Derived from the ledger. */
  seats_bots?: number
  seats_humans?: number
  granted_seats?: number
  sold_out_events?: number
  oversell_count?: number
  duplicate_count?: number
  trust_buckets?: number[]
  mode?: Mode
}

export interface LedgerEvent {
  timestamp: string | number
  user_id: string
  event_type: string
  reason_code: string | null
  trust_score?: number | null
  user_type?: UserType | null
  human_profile?: string | null
  ip?: string | null
  subnet?: string | null
  result?: string | null
}

export interface ResultRun {
  mode: Mode
  intensity: number
  bot_share: number
  human_win_rate?: number
  human_false_rejection_rate?: number
  seats_bots?: number
  seats_humans?: number
  total_seats?: number
  finished_at?: string

  // Measured by the API from the hash-chained ledger. These replaced a
  // hardcoded fixture, so the values are real observations per run.
  bot_advantage_ratio?: number
  humans_in_pool?: number
  bots_in_pool?: number
  precision?: number
  recall?: number
  oversell_count?: number
  duplicate_count?: number
  events_scanned?: number
  humans_denied?: number
  humans_lost_to_sold_out?: number
  bots_denied?: number
  attempts_limit_hit?: number
}

/** Full /results envelope: the runs plus chain-verification metadata. */
export interface ResultsResponse {
  runs: ResultRun[]
  measured: boolean
  chain_ok: boolean
  chain_checked: number
  chain_head?: string
  chain_error?: string
  total_seats: number
  current_mode: Mode
}

export interface PowChallenge {
  challenge: string
  difficulty: number
}

export interface JoinResponse extends PowChallenge {
  token: string
}

export interface SessionRaw {
  state?: string
  status?: string
  reason_code?: string | null
  claim_deadline?: string | number | null
  claim_expires_at?: string | number | null
  challenge?: string
  difficulty?: number
  [key: string]: unknown
}

export const getHealth = () => api<unknown>("/health")
export const getMetrics = () => api<Metrics>("/metrics")
export const getLedger = (limit = 50) => api<LedgerEvent[]>(`/ledger?limit=${limit}`)

/**
 * Ledger events that reached an actual system decision.
 *
 * The raw ledger is ~94% "sold_out" tail arriving after the pool closed, so any
 * newest-N window shows only that tail. This asks the server to filter, which
 * is the only way to see the events that reached a decision.
 */
export const getDecidedLedger = (limit = 400) =>
  api<LedgerEvent[]>(`/ledger?limit=${limit}&decided=1`)

// /results returns an envelope { runs, chain_ok, ... }, but every consumer
// (ModeCompare, AttackLevelChart, lib/results.ts) expects a plain array and
// calls .forEach/.map on it. Unwrapping here keeps the chain metadata available
// on the envelope while guaranteeing consumers always receive an array -- the
// mismatch previously threw and unmounted the whole Operator page.
export async function getResults(): Promise<ResultRun[]> {
  const body = await api<ResultsResponse | ResultRun[]>("/results")
  if (Array.isArray(body)) return body
  return Array.isArray(body?.runs) ? body.runs : []
}

/** Chain verification for the latest run, for an integrity panel. */
export async function getChainStatus() {
  const body = await api<ResultsResponse | ResultRun[]>("/results")
  if (Array.isArray(body)) return null
  return {
    ok: body.chain_ok,
    checked: body.chain_checked,
    head: body.chain_head,
    error: body.chain_error,
  }
}

export const join = () => api<JoinResponse>("/join", { method: "POST", body: "{}" })
export const verify = (token: string, nonce: string) =>
  api<SessionRaw>("/verify", {
    method: "POST",
    headers: authHeader(token),
    body: JSON.stringify({ token, nonce }),
  })
export const getSession = (token: string) =>
  api<SessionRaw>(`/session?token=${encodeURIComponent(token)}`, { headers: authHeader(token) })
export const claim = (token: string) =>
  api<SessionRaw>("/claim", {
    method: "POST",
    headers: authHeader(token),
    body: JSON.stringify({ token }),
  })

export const setMode = (mode: Mode) =>
  api<unknown>("/admin/mode", { method: "POST", body: JSON.stringify({ mode }) })
export const setConfig = (configJson: string) =>
  api<unknown>("/admin/config", { method: "POST", body: JSON.stringify({ config_json: configJson }) })
export const resetAll = () => api<unknown>("/admin/reset", { method: "POST", body: "{}" })

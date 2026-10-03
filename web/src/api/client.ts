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
export const getResults = () => api<ResultRun[]>("/results")

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

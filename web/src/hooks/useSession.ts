/**
 * File: web/src/hooks/useSession.ts
 * Purpose: User-side session lifecycle. Persists token + PoW challenge in
 *          localStorage, runs the PoW worker, calls /join, /verify, /claim,
 *          and restores state from GET /session after refresh/reconnect.
 */
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ApiError,
  claim as claimApi,
  getSession,
  join as joinApi,
  verify as verifyApi,
  type PowChallenge,
  type SessionRaw,
} from "@/api/client"

const TOKEN_KEY = "fairdrop.token"
const CHALLENGE_KEY = "fairdrop.challenge"
const VERIFIED_KEY = "fairdrop.verified"

export type Phase =
  | "idle"
  | "verifying"
  | "waiting"
  | "in_draw"
  | "won"
  | "claimed"
  | "not_selected"
  | "rejected"

export interface PowState {
  attempts: number
  expected: number
  hashRate: number
}

const listeners = new Set<() => void>()
function notify() {
  listeners.forEach((l) => l())
}
function subscribe(l: () => void) {
  listeners.add(l)
  window.addEventListener("storage", l)
  return () => {
    listeners.delete(l)
    window.removeEventListener("storage", l)
  }
}

function readStoredToken(): string | null {
  try {
    const raw = localStorage.getItem(TOKEN_KEY)
    if (!raw || raw === "undefined" || raw === "null" || raw.trim() === "") {
      return null
    }
    return raw
  } catch {
    return null
  }
}

function readChallenge(): PowChallenge | null {
  try {
    const raw = localStorage.getItem(CHALLENGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as PowChallenge
    if (!parsed || typeof parsed.difficulty !== "number") return null
    return parsed
  } catch {
    return null
  }
}

function mapState(raw: SessionRaw): { phase: Phase; reason: string | null } {
  const s = String(raw.state ?? raw.status ?? "").toLowerCase()
  const reason = raw.reason_code ?? null
  if (s.startsWith("rejected") || s === "duplicate_claim" || s === "sold_out" || s === "expired")
    return { phase: "rejected", reason: reason ?? s }
  if (s === "claimed" || s === "seat_granted" || s === "allocated") return { phase: "claimed", reason }
  if (s === "won" || s === "selected") return { phase: "won", reason }
  if (s.startsWith("not_selected")) return { phase: "not_selected", reason }
  if (s === "in_draw" || s === "draw" || s === "drawing") return { phase: "in_draw", reason }
  if (s === "pending_pow" || s === "unverified" || s === "joined") return { phase: "verifying", reason }
  return { phase: "waiting", reason }
}

export function useSession() {
  const qc = useQueryClient()
  const token = useSyncExternalStore(subscribe, readStoredToken)
  const verified = useSyncExternalStore(subscribe, () => localStorage.getItem(VERIFIED_KEY) === "1")
  const [pow, setPow] = useState<PowState | null>(null)
  const [error, setError] = useState<string | null>(null)
  const workerRef = useRef<Worker | null>(null)

  const session = useQuery({
    queryKey: ["session", token],
    queryFn: () => getSession(token as string),
    enabled: !!token && verified,
    refetchInterval: (q) => {
      const data = q.state.data
      if (!data) return 1000
      const p = mapState(data).phase
      return p === "waiting" || p === "in_draw" || p === "won" ? 1000 : false
    },
    retry: (count, err) => !(err instanceof ApiError && (err.status === 401 || err.status === 404)) && count < 2,
  })

  const clear = useCallback(() => {
    workerRef.current?.terminate()
    workerRef.current = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(CHALLENGE_KEY)
    localStorage.removeItem(VERIFIED_KEY)
    setPow(null)
    setError(null)
    qc.removeQueries({ queryKey: ["session"] })
    notify()
  }, [qc])

  const verifyMut = useMutation({
    mutationFn: ({ t, nonce }: { t: string; nonce: string }) => verifyApi(t, nonce),
    onSuccess: (data, { t }) => {
      localStorage.setItem(VERIFIED_KEY, "1")
      localStorage.removeItem(CHALLENGE_KEY)
      if (data && (data.state || data.status)) qc.setQueryData(["session", t], data)
      setPow(null)
      notify()
    },
    onError: (e: Error) => {
      setPow(null)
      setError(e.message)
      if (e instanceof ApiError && e.status >= 400 && e.status < 500) {
        localStorage.setItem(VERIFIED_KEY, "1")
        localStorage.removeItem(CHALLENGE_KEY)
        notify()
      }
    },
  })

  const solve = useCallback(
    (t: string, ch: PowChallenge) => {
      const diff = typeof ch.difficulty === "number" ? ch.difficulty : 4
      workerRef.current?.terminate()
      const worker = new Worker(new URL("../lib/pow.worker.ts", import.meta.url), { type: "module" })
      workerRef.current = worker
      setPow({ attempts: 0, expected: 2 ** diff, hashRate: 0 })
      worker.postMessage({ challenge: ch.challenge || "solve_me", difficulty: diff })
      worker.onmessage = (ev) => {
        const msg = ev.data
        if (msg.type === "progress") {
          setPow({ attempts: msg.attempts, expected: 2 ** diff, hashRate: msg.hashRate })
        } else if (msg.type === "done") {
          worker.terminate()
          workerRef.current = null
          setPow({ attempts: msg.attempts, expected: msg.attempts, hashRate: 0 })
          verifyMut.mutate({ t, nonce: msg.nonce })
        }
      }
      worker.onerror = () => {
        setPow(null)
        setError("Verification failed to start in this browser")
      }
    },
    [verifyMut],
  )

  const joinMut = useMutation({
    mutationFn: joinApi,
    onMutate: () => setError(null),
    onSuccess: (raw) => {
      const data = raw as unknown as Record<string, unknown>
      const tok = (data.token ?? data.session_token ?? "") as string
      const ch = (data.challenge ?? data.pow_challenge ?? "solve_me") as string
      const diff = typeof data.difficulty === "number" ? data.difficulty : 4

      if (!tok) {
        setError("Invalid response from server")
        return
      }

      localStorage.setItem(TOKEN_KEY, tok)
      localStorage.setItem(CHALLENGE_KEY, JSON.stringify({ challenge: ch, difficulty: diff }))
      localStorage.removeItem(VERIFIED_KEY)
      notify()
      solve(tok, { challenge: ch, difficulty: diff })
    },
    onError: (e: Error) => setError(e.message),
  })

  const claimMut = useMutation({
    mutationFn: () => claimApi(token as string),
    onSuccess: (data) => {
      if (data && (data.state || data.status)) qc.setQueryData(["session", token], data)
      qc.invalidateQueries({ queryKey: ["session", token] })
    },
    onError: (e: Error) => setError(e.message),
  })

  // Resume or clean up
  useEffect(() => {
    if (token) {
      if (token === "undefined" || token === "null") {
        clear()
        return
      }
      if (!verified && !workerRef.current && !verifyMut.isPending) {
        const ch = readChallenge()
        if (ch) {
          solve(token, ch)
        } else {
          clear()
        }
      }
    }
  }, [token, verified, solve, verifyMut.isPending, clear])

  useEffect(
    () => () => {
      workerRef.current?.terminate()
      workerRef.current = null
    },
    [],
  )

  let phase: Phase = "idle"
  let reason: string | null = null
  if (token && token !== "undefined" && token !== "null") {
    if (!verified) phase = "verifying"
    else if (session.data) ({ phase, reason } = mapState(session.data))
    else phase = "waiting"
  }

  const deadlineRaw = session.data?.claim_deadline ?? session.data?.claim_expires_at ?? null
  const claimDeadline =
    deadlineRaw == null
      ? null
      : typeof deadlineRaw === "number"
        ? deadlineRaw < 1e12
          ? deadlineRaw * 1000
          : deadlineRaw
        : Date.parse(deadlineRaw) || null

  const sessionLost = session.error instanceof ApiError && (session.error.status === 401 || session.error.status === 404)
  const stuckVerifying = phase === "verifying" && !pow && !verifyMut.isPending && !readChallenge()

  return {
    token,
    phase,
    reason,
    pow,
    claimDeadline,
    error: error ?? (session.error && !sessionLost ? session.error.message : null),
    sessionLost: sessionLost || stuckVerifying,
    joining: joinMut.isPending,
    verifying: verifyMut.isPending,
    claiming: claimMut.isPending,
    join: () => joinMut.mutate(),
    claim: () => claimMut.mutate(),
    leave: clear,
  }
}

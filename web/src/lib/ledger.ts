/**
 * File: web/src/lib/ledger.ts
 * Purpose: Ledger event colour mapping (FRONTEND_SPEC §5.2.9) and plain
 *          operator-language labels for event types and reason codes.
 */

export type Tone = "ok" | "warn" | "bad" | "neutral"

export function eventTone(eventType: string): Tone {
  if (eventType === "seat_granted") return "ok"
  if (
    eventType === "rejected_low_trust" ||
    eventType.startsWith("rejected_pow_") ||
    eventType.startsWith("rejected_ratelimit_")
  )
    return "warn"
  if (eventType === "duplicate_claim" || eventType === "sold_out") return "bad"
  return "neutral"
}

export const toneText: Record<Tone, string> = {
  ok: "text-ok",
  warn: "text-warn",
  bad: "text-bad",
  neutral: "text-muted-foreground",
}

export const toneDot: Record<Tone, string> = {
  ok: "bg-ok",
  warn: "bg-warn",
  bad: "bg-bad",
  neutral: "bg-muted-foreground/50",
}

const LABELS: Record<string, string> = {
  seat_granted: "Seat granted",
  accepted_pool: "Accepted to pool",
  selected: "Selected in draw",
  not_selected_draw: "Not selected in draw",
  rejected_low_trust: "Rejected: trust score too low",
  rejected_pow_too_fast: "Rejected: PoW solved too fast",
  rejected_pow_invalid: "Rejected: PoW answer invalid",
  rejected_pow_expired: "Rejected: PoW challenge expired",
  rejected_ratelimit_ip: "Rejected: IP rate limit",
  rejected_ratelimit_user: "Rejected: user rate limit",
  duplicate_claim: "Duplicate claim blocked",
  sold_out: "Sold out",
}

export function label(code: string | null | undefined): string {
  if (!code) return "—"
  if (LABELS[code]) return LABELS[code]
  const words = code.replace(/_/g, " ")
  if (code.startsWith("rejected_")) return `Rejected: ${words.slice("rejected ".length)}`
  return words.charAt(0).toUpperCase() + words.slice(1)
}

export function deriveResult(eventType: string): string {
  const tone = eventTone(eventType)
  if (tone === "ok") return "Granted"
  if (tone === "warn") return "Rejected"
  if (tone === "bad") return "Blocked"
  return "In progress"
}

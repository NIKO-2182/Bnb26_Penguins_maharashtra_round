/**
 * File: web/src/lib/utils.ts
 * Purpose: Class merging and number formatting helpers shared by all panels.
 */
import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function fmtPct(v: number | null | undefined, digits = 1): string {
  if (v == null || Number.isNaN(v)) return "—"
  return `${(v * 100).toFixed(digits)}%`
}

export function fmtInt(v: number | null | undefined): string {
  if (v == null || Number.isNaN(v)) return "—"
  return Math.round(v).toLocaleString("en-US")
}

export function fmtTime(ts: string | number): string {
  const d = typeof ts === "number" ? new Date(ts < 1e12 ? ts * 1000 : ts) : new Date(ts)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleTimeString("en-GB", { hour12: false }) + "." + String(d.getMilliseconds()).padStart(3, "0")
}

export function shortId(id: string): string {
  return id.length > 10 ? `${id.slice(0, 8)}…` : id
}

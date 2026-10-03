/**
 * File: web/src/api/sim.ts
 * Purpose: Typed wrappers for the simulator control service (sim:8090),
 *          reached through the /sim proxy.
 */
import { request } from "./client"

const SIM_BASE = import.meta.env.VITE_SIM_BASE ?? "/sim"

export const BOT_PROFILES = ["naive", "solver", "distributed", "retry"] as const
export type BotProfile = (typeof BOT_PROFILES)[number]

export interface SimConfig {
  bots: number
  rate: number
  retries: number
  ip_spread: number
  profile: BotProfile
  humans: number
}

export interface SimStatus {
  running: boolean
  started_at?: string | number | null
  elapsed_s?: number | null
  config?: Partial<SimConfig>
}

export const runSim = (config: SimConfig) =>
  request<SimStatus>(SIM_BASE, "/run", { method: "POST", body: JSON.stringify(config) })
export const stopSim = () => request<SimStatus>(SIM_BASE, "/stop", { method: "POST", body: "{}" })
export const getSimStatus = () => request<SimStatus>(SIM_BASE, "/status")

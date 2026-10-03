/**
 * File: web/src/store/adminStore.ts
 * Purpose: Zustand store for operator-side mode/config state only:
 *          last mode the operator requested, config JSON draft, simulator
 *          settings, and a reset epoch that clears client-side series.
 */
import { create } from "zustand"
import type { Mode } from "@/api/client"
import type { SimConfig } from "@/api/sim"

interface AdminState {
  requestedMode: Mode | null
  configJson: string
  sim: SimConfig
  resetEpoch: number
  setRequestedMode: (m: Mode) => void
  setConfigJson: (s: string) => void
  setSim: (patch: Partial<SimConfig>) => void
  bumpReset: () => void
}

export const useAdminStore = create<AdminState>((set) => ({
  requestedMode: null,
  configJson: "{\n  \n}",
  sim: { bots: 500, rate: 200, retries: 2, ip_spread: 20, profile: "naive", humans: 200 },
  resetEpoch: 0,
  setRequestedMode: (requestedMode) => set({ requestedMode }),
  setConfigJson: (configJson) => set({ configJson }),
  setSim: (patch) => set((s) => ({ sim: { ...s.sim, ...patch } })),
  bumpReset: () => set((s) => ({ resetEpoch: s.resetEpoch + 1 })),
}))

/**
 * File: web/src/hooks/useMetrics.ts
 * Purpose: Polls GET /metrics every second. Also exposes useResults
 *          (GET /results) and useSimStatus (sim GET /status).
 */
import { useQuery } from "@tanstack/react-query"
import { getMetrics, getResults } from "@/api/client"
import { getSimStatus } from "@/api/sim"

export function useMetrics() {
  return useQuery({
    queryKey: ["metrics"],
    queryFn: getMetrics,
    refetchInterval: 1000,
    retry: false,
  })
}

export function useResults() {
  return useQuery({
    queryKey: ["results"],
    queryFn: getResults,
    refetchInterval: 5000,
    retry: false,
  })
}

export function useSimStatus() {
  return useQuery({
    queryKey: ["sim-status"],
    queryFn: getSimStatus,
    refetchInterval: 1000,
    retry: false,
  })
}

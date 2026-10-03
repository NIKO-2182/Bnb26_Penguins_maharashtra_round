/**
 * File: web/src/components/SimControls.tsx
 * Purpose: Simulator control panel (bots, rate, retries, IP spread, profile,
 *          humans) and the header Run / Stop button with elapsed time.
 */
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Play, Square } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Slider } from "@/components/ui/slider"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Panel, QueryError } from "@/components/Panel"
import { BOT_PROFILES, runSim, stopSim, type BotProfile, type SimConfig, type SimStatus } from "@/api/sim"
import { useSimStatus } from "@/hooks/useMetrics"
import { useAdminStore } from "@/store/adminStore"
import { fmtInt } from "@/lib/utils"

function elapsedSeconds(s: SimStatus | undefined): number | null {
  if (!s?.running) return null
  if (s.elapsed_s != null) return s.elapsed_s
  if (s.started_at == null) return null
  const start =
    typeof s.started_at === "number"
      ? s.started_at < 1e12
        ? s.started_at * 1000
        : s.started_at
      : Date.parse(s.started_at)
  return Number.isNaN(start) ? null : Math.max(0, (Date.now() - start) / 1000)
}

function fmtElapsed(sec: number) {
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`
}

function useSimRun() {
  const qc = useQueryClient()
  const status = useSimStatus()
  const sim = useAdminStore((s) => s.sim)
  const done = () => qc.invalidateQueries({ queryKey: ["sim-status"] })
  const run = useMutation({ mutationFn: (c: SimConfig) => runSim(c), onSettled: done })
  const stop = useMutation({ mutationFn: stopSim, onSettled: done })
  return { status, sim, run, stop }
}

export function SimRunButton() {
  const { status, sim, run, stop } = useSimRun()
  const running = status.data?.running ?? false
  const elapsed = elapsedSeconds(status.data)
  const unreachable = status.isError

  return (
    <div className="flex items-center gap-2">
      {running && elapsed != null && (
        <span className="num flex items-center gap-1.5 text-xs text-muted-foreground" aria-live="polite">
          <span aria-hidden className="size-1.5 animate-pulse rounded-full bg-ok" />
          {fmtElapsed(elapsed)}
        </span>
      )}
      {running ? (
        <Button size="sm" variant="secondary" onClick={() => stop.mutate()} disabled={stop.isPending}>
          <Square data-icon="inline-start" />
          Stop
        </Button>
      ) : (
        <Button
          size="sm"
          onClick={() => run.mutate(sim)}
          disabled={run.isPending || unreachable}
          title={unreachable ? "Simulator unreachable" : undefined}
        >
          <Play data-icon="inline-start" />
          Run simulation
        </Button>
      )}
    </div>
  )
}

function SliderField({
  id,
  label,
  value,
  min,
  max,
  step = 1,
  unit,
  onChange,
  disabled,
}: {
  id: string
  label: string
  value: number
  min: number
  max: number
  step?: number
  unit?: string
  onChange: (n: number) => void
  disabled?: boolean
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <Label id={id} className="text-xs font-normal text-muted-foreground">
          {label}
        </Label>
        <span className="num text-xs">
          {fmtInt(value)}
          {unit && <span className="text-muted-foreground">{unit}</span>}
        </span>
      </div>
      <Slider
        aria-labelledby={id}
        value={[value]}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        onValueChange={(v) => onChange(Array.isArray(v) ? v[0] : v)}
      />
    </div>
  )
}

export function SimPanel() {
  const { status } = useSimRun()
  const sim = useAdminStore((s) => s.sim)
  const setSim = useAdminStore((s) => s.setSim)
  const running = status.data?.running ?? false

  return (
    <Panel title="Simulator" actions={<SimRunButton />}>
      {status.isError && (
        <div className="mb-3">
          <QueryError error={status.error} what="Simulator" />
        </div>
      )}
      <div className="grid gap-x-6 gap-y-4 sm:grid-cols-2">
        <SliderField id="sim-bots" label="Bots" value={sim.bots} min={0} max={5000} step={50} onChange={(bots) => setSim({ bots })} disabled={running} />
        <SliderField id="sim-humans" label="Humans" value={sim.humans} min={0} max={2000} step={10} onChange={(humans) => setSim({ humans })} disabled={running} />
        <SliderField id="sim-rate" label="Rate" value={sim.rate} min={1} max={2000} step={10} unit=" req/s" onChange={(rate) => setSim({ rate })} disabled={running} />
        <SliderField id="sim-retries" label="Retries" value={sim.retries} min={0} max={10} onChange={(retries) => setSim({ retries })} disabled={running} />
        <SliderField id="sim-ips" label="IP spread" value={sim.ip_spread} min={1} max={500} unit=" IPs" onChange={(ip_spread) => setSim({ ip_spread })} disabled={running} />
        <div className="flex flex-col gap-2">
          <Label id="sim-profile" className="text-xs font-normal text-muted-foreground">
            Bot profile
          </Label>
          <Select
            items={BOT_PROFILES.map((p) => ({ value: p, label: p }))}
            value={sim.profile}
            onValueChange={(v) => v && setSim({ profile: v as BotProfile })}
            disabled={running}
          >
            <SelectTrigger size="sm" aria-labelledby="sim-profile" className="w-full capitalize">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {BOT_PROFILES.map((p) => (
                <SelectItem key={p} value={p} className="capitalize">
                  {p}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
    </Panel>
  )
}

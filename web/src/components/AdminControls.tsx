/**
 * File: web/src/components/AdminControls.tsx
 * Purpose: Operator admin actions: FCFS/Fair mode switch, config JSON editor,
 *          and reset with confirmation.
 */
import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Panel } from "@/components/Panel"
import { resetAll, setConfig, setMode, type Mode } from "@/api/client"
import { useMetrics } from "@/hooks/useMetrics"
import { useAdminStore } from "@/store/adminStore"
import { cn } from "@/lib/utils"

export function ModeSwitch() {
  const qc = useQueryClient()
  const { data } = useMetrics()
  const requested = useAdminStore((s) => s.requestedMode)
  const setRequested = useAdminStore((s) => s.setRequestedMode)
  const mut = useMutation({
    mutationFn: setMode,
    onMutate: (m: Mode) => setRequested(m),
    onSettled: () => qc.invalidateQueries({ queryKey: ["metrics"] }),
  })
  const current = data?.mode ?? requested

  return (
    <div role="radiogroup" aria-label="Allocation mode" className="flex rounded-md border p-0.5">
      {(["fcfs", "fair"] as const).map((m) => {
        const active = current === m
        return (
          <button
            key={m}
            type="button"
            role="radio"
            aria-checked={active}
            disabled={mut.isPending}
            onClick={() => !active && mut.mutate(m)}
            className={cn(
              "num h-6 rounded-[4px] px-2.5 text-[11px] uppercase transition-colors disabled:opacity-60",
              active ? "bg-secondary text-foreground" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {m}
          </button>
        )
      })}
      {mut.isError && <span className="sr-only">Mode change failed: {mut.error.message}</span>}
    </div>
  )
}

export function ResetButton() {
  const qc = useQueryClient()
  const bumpReset = useAdminStore((s) => s.bumpReset)
  const [open, setOpen] = useState(false)
  const mut = useMutation({
    mutationFn: resetAll,
    onSuccess: () => {
      bumpReset()
      setOpen(false)
      qc.invalidateQueries()
    },
  })

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button size="sm" variant="ghost" />}>
        <RotateCcw data-icon="inline-start" />
        Reset
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Reset the drop?</AlertDialogTitle>
          <AlertDialogDescription>
            This clears the pool, seats, and ledger on the server. Charts restart from zero. Results from finished runs
            are kept.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {mut.isError && <p className="text-xs text-bad">{mut.error.message}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={() => mut.mutate()} disabled={mut.isPending}>
            {mut.isPending ? "Resetting" : "Reset"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

export function ConfigEditor() {
  const json = useAdminStore((s) => s.configJson)
  const setJson = useAdminStore((s) => s.setConfigJson)
  const mut = useMutation({ mutationFn: setConfig })

  let parseError: string | null = null
  try {
    JSON.parse(json)
  } catch (e) {
    parseError = (e as Error).message
  }

  return (
    <Panel
      title="Config"
      actions={
        <Button
          size="xs"
          variant="secondary"
          disabled={!!parseError || mut.isPending}
          onClick={() => mut.mutate(json)}
        >
          {mut.isPending ? "Applying" : "Apply"}
        </Button>
      }
    >
      <label htmlFor="config-json" className="sr-only">
        Config JSON
      </label>
      <Textarea
        id="config-json"
        value={json}
        onChange={(e) => {
          setJson(e.target.value)
          mut.reset()
        }}
        spellCheck={false}
        className="num min-h-32 resize-y text-xs"
      />
      <p
        className={cn(
          "num mt-2 min-h-4 text-[11px]",
          parseError || mut.isError ? "text-bad" : mut.isSuccess ? "text-ok" : "text-muted-foreground",
        )}
        aria-live="polite"
      >
        {parseError
          ? `Invalid JSON: ${parseError}`
          : mut.isError
            ? mut.error.message
            : mut.isSuccess
              ? "Applied"
              : "POST /admin/config"}
      </p>
    </Panel>
  )
}

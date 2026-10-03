/**
 * File: web/src/components/StatusCard.tsx
 * Purpose: The single user-facing status card. Shows exactly one state at a
 *          time: Join, Verifying, Waiting, In draw, Won, Claimed, Not selected,
 *          Rejected.
 */
import { useEffect, useState, type ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { PowProgress } from "@/components/PowProgress"
import type { useSession } from "@/hooks/useSession"
import { label } from "@/lib/ledger"
import { cn } from "@/lib/utils"

type Session = ReturnType<typeof useSession>

function useCountdown(deadline: number | null) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (deadline == null) return
    const id = setInterval(() => setNow(Date.now()), 250)
    return () => clearInterval(id)
  }, [deadline])
  return deadline == null ? null : Math.max(0, Math.ceil((deadline - now) / 1000))
}

function Status({ tone, title, children }: { tone: "ok" | "warn" | "bad" | "neutral"; title: string; children?: ReactNode }) {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <span
          aria-hidden
          className={cn(
            "size-2 rounded-full",
            tone === "ok" && "bg-ok",
            tone === "warn" && "bg-warn",
            tone === "bad" && "bg-bad",
            tone === "neutral" && "bg-muted-foreground",
          )}
        />
        <h2 className="text-base font-medium">{title}</h2>
      </div>
      {children}
    </div>
  )
}

function Muted({ children }: { children: ReactNode }) {
  return <p className="text-sm leading-relaxed text-muted-foreground">{children}</p>
}

export function StatusCard({ s }: { s: Session }) {
  const remaining = useCountdown(s.phase === "won" ? s.claimDeadline : null)

  let body: ReactNode
  switch (s.phase) {
    case "idle":
      body = (
        <Status tone="neutral" title="Join the drop">
          <Muted>{"You'll get a place in the pool. Seats are given out by a fair draw, not by who clicks fastest."}</Muted>
          <Button onClick={s.join} disabled={s.joining} className="self-start">
            {s.joining ? "Joining" : "Join"}
          </Button>
        </Status>
      )
      break
    case "verifying":
      body = <PowProgress pow={s.pow} submitting={s.verifying} />
      break
    case "waiting":
      body = (
        <Status tone="neutral" title="You're in the pool">
          <Muted>Waiting for the draw to start. You can leave this page open or come back later.</Muted>
        </Status>
      )
      break
    case "in_draw":
      body = (
        <Status tone="neutral" title="Draw in progress">
          <Muted>Seats are being drawn now. Your result will appear here.</Muted>
        </Status>
      )
      break
    case "won":
      body = (
        <Status tone="ok" title="You won a seat">
          <Muted>
            {remaining != null ? (
              <>
                Claim it within <span className="num text-foreground">{remaining}s</span> or it goes to the next person.
              </>
            ) : (
              "Claim it now or it goes to the next person."
            )}
          </Muted>
          <Button onClick={s.claim} disabled={s.claiming || remaining === 0} className="self-start">
            {s.claiming ? "Claiming" : "Claim seat"}
          </Button>
        </Status>
      )
      break
    case "claimed":
      body = (
        <Status tone="ok" title="Seat confirmed">
          <Muted>Your seat is yours. No further action needed.</Muted>
        </Status>
      )
      break
    case "not_selected":
      body = (
        <Status tone="neutral" title="Not selected">
          <Muted>The draw didn&apos;t pick you this time. Every person in the pool had the same chance.</Muted>
        </Status>
      )
      break
    case "rejected":
      body = (
        <Status tone="warn" title="Entry rejected">
          <Muted>{s.reason ? label(s.reason) : "Your entry could not be accepted."}</Muted>
          {s.reason && <p className="num text-xs text-muted-foreground">{s.reason}</p>}
        </Status>
      )
      break
  }

  return (
    <section aria-live="polite" className="rounded-lg border bg-card p-6">
      {body}
      {s.error && <p className="mt-4 text-xs text-bad">{s.error}</p>}
      {s.sessionLost && (
        <div className="mt-4 flex items-center justify-between gap-3 border-t pt-4">
          <p className="text-xs text-muted-foreground">Your session could not be found.</p>
          <Button size="sm" variant="secondary" onClick={s.leave}>
            Start over
          </Button>
        </div>
      )}
    </section>
  )
}

/**
 * File: web/src/components/LedgerDrawer.tsx
 * Purpose: Side drawer explaining one ledger event in operator language:
 *          user, trust score, outcome, and the plain reason.
 */
import type { ReactNode } from "react"
import type { KeyedEvent } from "@/hooks/useLedger"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { deriveResult, eventTone, label, toneDot, toneText } from "@/lib/ledger"
import { cn, fmtTime } from "@/lib/utils"

function Field({ name, children, mono }: { name: string; children: ReactNode; mono?: boolean }) {
  return (
    <div className="grid grid-cols-[7rem_1fr] gap-3 border-t py-2 text-xs">
      <dt className="text-muted-foreground">{name}</dt>
      <dd className={cn("min-w-0 break-all", mono && "num")}>{children}</dd>
    </div>
  )
}

export function LedgerDrawer({ event, onClose }: { event: KeyedEvent | null; onClose: () => void }) {
  const tone = event ? eventTone(event.event_type) : "neutral"
  return (
    <Sheet open={event != null} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full gap-0 sm:max-w-md">
        {event && (
          <>
            <SheetHeader className="border-b">
              <SheetTitle className={cn("flex items-center gap-2", toneText[tone])}>
                <span aria-hidden className={cn("size-2 rounded-full", toneDot[tone])} />
                {label(event.event_type)}
              </SheetTitle>
              <SheetDescription className="num text-xs">{fmtTime(event.timestamp)}</SheetDescription>
            </SheetHeader>
            <dl className="px-4 pb-4">
              <Field name="User" mono>
                {event.user_id}
              </Field>
              <Field name="User type">{event.user_type ?? "Not reported"}</Field>
              {event.human_profile && <Field name="Profile">{event.human_profile}</Field>}
              {event.ip && <Field name="IP" mono>{event.ip}</Field>}
              {event.subnet && <Field name="Subnet" mono>{event.subnet}</Field>}
              <Field name="Trust score" mono>
                {event.trust_score != null ? event.trust_score.toFixed(3) : "—"}
              </Field>
              <Field name="Result">{event.result ?? deriveResult(event.event_type)}</Field>
              <Field name="Reason">
                {event.reason_code ? (
                  <>
                    {label(event.reason_code)}
                    <span className="num mt-0.5 block text-[11px] text-muted-foreground">{event.reason_code}</span>
                  </>
                ) : (
                  "—"
                )}
              </Field>
              <Field name="Event type" mono>
                {event.event_type}
              </Field>
            </dl>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}

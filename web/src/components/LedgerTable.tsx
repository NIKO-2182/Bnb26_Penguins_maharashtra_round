/**
 * File: web/src/components/LedgerTable.tsx
 * Purpose: Live tail of GET /ledger. Newest first, rows slide in, filterable
 *          by reason code and user type. Clicking a row opens LedgerDrawer.
 */
import { useMemo, useState } from "react"
import type { KeyedEvent } from "@/hooks/useLedger"
import { useLedger } from "@/hooks/useLedger"
import { Panel, QueryError, EmptyState } from "@/components/Panel"
import { LedgerDrawer } from "@/components/LedgerDrawer"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { eventTone, label, toneDot, toneText } from "@/lib/ledger"
import { cn, fmtTime, shortId } from "@/lib/utils"

const ALL = "__all__"

function FilterSelect({
  name,
  value,
  options,
  onChange,
}: {
  name: string
  value: string
  options: string[]
  onChange: (v: string) => void
}) {
  const items = [{ value: ALL, label: `All ${name}s` }, ...options.map((o) => ({ value: o, label: label(o) }))]
  return (
    <Select items={items} value={value} onValueChange={(v) => onChange((v as string) ?? ALL)}>
      <SelectTrigger size="sm" aria-label={`Filter by ${name}`} className="h-6 min-w-32 text-[11px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {items.map((i) => (
          <SelectItem key={i.value} value={i.value} className="text-xs">
            {i.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function LedgerTable() {
  const { data, error, isLoading } = useLedger(50)
  const [userType, setUserType] = useState(ALL)
  const [reason, setReason] = useState(ALL)
  const [selected, setSelected] = useState<KeyedEvent | null>(null)

  const { userTypes, reasons } = useMemo(() => {
    const u = new Set<string>()
    const r = new Set<string>()
    data?.forEach((e) => {
      if (e.user_type) u.add(e.user_type)
      if (e.reason_code) r.add(e.reason_code)
    })
    return { userTypes: [...u].sort(), reasons: [...r].sort() }
  }, [data])

  const rows = (data ?? []).filter(
    (e) => (userType === ALL || e.user_type === userType) && (reason === ALL || e.reason_code === reason),
  )

  let content
  if (isLoading) content = <EmptyState>Loading</EmptyState>
  else if (error) content = <QueryError error={error} what="Ledger" />
  else if (!data?.length) content = <EmptyState>No events yet</EmptyState>
  else if (!rows.length) content = <EmptyState>No events match the filters</EmptyState>
  else
    content = (
      <table className="w-full table-fixed text-xs">
        <thead className="sticky top-0 bg-card">
          <tr className="text-left text-[11px] uppercase tracking-wider text-muted-foreground">
            <th scope="col" className="w-28 px-3 py-1.5 font-medium">Time</th>
            <th scope="col" className="w-28 px-3 py-1.5 font-medium">User</th>
            <th scope="col" className="px-3 py-1.5 font-medium">Event</th>
            <th scope="col" className="hidden px-3 py-1.5 font-medium md:table-cell">Reason</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((e) => {
            const tone = eventTone(e.event_type)
            return (
              <tr
                key={e.key}
                tabIndex={0}
                onClick={() => setSelected(e)}
                onKeyDown={(ev) => {
                  if (ev.key === "Enter" || ev.key === " ") {
                    ev.preventDefault()
                    setSelected(e)
                  }
                }}
                aria-label={`${label(e.event_type)} for user ${e.user_id}`}
                className="row-in cursor-pointer border-t outline-none hover:bg-accent focus-visible:bg-accent"
              >
                <td className="num truncate px-3 py-1.5 text-muted-foreground">{fmtTime(e.timestamp)}</td>
                <td className="num truncate px-3 py-1.5" title={e.user_id}>
                  {shortId(e.user_id)}
                </td>
                <td className={cn("truncate px-3 py-1.5", toneText[tone])}>
                  <span className="flex items-center gap-2">
                    <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", toneDot[tone])} />
                    <span className="truncate">{label(e.event_type)}</span>
                  </span>
                </td>
                <td className="num hidden truncate px-3 py-1.5 text-muted-foreground md:table-cell">
                  {e.reason_code ?? "—"}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    )

  return (
    <>
      <Panel
        title="Ledger"
        actions={
          <div className="flex items-center gap-1.5">
            {userTypes.length > 0 && (
              <FilterSelect name="user type" value={userType} options={userTypes} onChange={setUserType} />
            )}
            <FilterSelect name="reason" value={reason} options={reasons} onChange={setReason} />
          </div>
        }
        bodyClassName="h-96 overflow-y-auto p-0"
      >
        {content}
      </Panel>
      <LedgerDrawer event={selected} onClose={() => setSelected(null)} />
    </>
  )
}

/**
 * File: web/src/pages/UserView.tsx
 * Purpose: Calm waiting-room screen for a real person joining the drop.
 */
import { Link } from "react-router-dom"
import { StatusCard } from "@/components/StatusCard"
import { useSession } from "@/hooks/useSession"

export default function UserView() {
  const s = useSession()
  return (
    <div className="flex min-h-svh flex-col">
      <header className="flex h-12 items-center justify-between border-b px-4">
        <span className="text-sm font-medium">Fair Drop</span>
        <Link to="/operator" className="text-xs text-muted-foreground hover:text-foreground">
          Operator
        </Link>
      </header>
      <main className="flex flex-1 items-start justify-center px-4 pt-[18vh]">
        <div className="flex w-full max-w-sm flex-col gap-4">
          <StatusCard s={s} />
          {s.token && (
            <p className="num text-center text-[11px] text-muted-foreground">
              Session {s.token.slice(0, 8)}
            </p>
          )}
        </div>
      </main>
    </div>
  )
}

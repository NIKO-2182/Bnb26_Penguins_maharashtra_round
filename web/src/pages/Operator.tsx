/**
 * File: web/src/pages/Operator.tsx
 * Purpose: Operator dashboard layout: sidebar, header actions, counters,
 *          hero stat, charts, live ledger, invariants, sim and admin panels.
 */
import { NavLink } from "react-router-dom"
import { Activity, Users } from "lucide-react"
import { LiveCounters } from "@/components/LiveCounters"
import { HeroStat } from "@/components/HeroStat"
import { OutcomeChart } from "@/components/OutcomeChart"
import { HumanCost } from "@/components/HumanCost"
import { ModeCompare } from "@/components/ModeCompare"
import { AttackLevelChart } from "@/components/AttackLevelChart"
import { TrustHistogram } from "@/components/TrustHistogram"
import { InvariantPanel } from "@/components/InvariantPanel"
import { LedgerTable } from "@/components/LedgerTable"
import { SimPanel, SimRunButton } from "@/components/SimControls"
import { ConfigEditor, ModeSwitch, ResetButton } from "@/components/AdminControls"
import { ConfusionMatrixPanel } from "@/components/ConfusionMatrixPanel"
import { useMetrics } from "@/hooks/useMetrics"
import { cn } from "@/lib/utils"

const NAV = [
  { to: "/operator", label: "Operator", icon: Activity },
  { to: "/", label: "User view", icon: Users },
]

function Sidebar() {
  return (
    <nav aria-label="Main" className="hidden w-44 shrink-0 flex-col border-r md:flex">
      <div className="flex h-12 items-center border-b px-4 text-sm font-medium">Fair Drop</div>
      <ul className="flex flex-col gap-0.5 p-2">
        {NAV.map(({ to, label, icon: Icon }) => (
          <li key={to}>
            <NavLink
              to={to}
              end
              className={({ isActive }) =>
                cn(
                  "flex h-7 items-center gap-2 rounded-md px-2 text-xs",
                  isActive ? "bg-secondary text-foreground" : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              <Icon className="size-3.5" aria-hidden />
              {label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}

function ConnectionDot() {
  const { isError, isSuccess } = useMetrics()
  const state = isError ? "API unreachable" : isSuccess ? "Live" : "Connecting"
  return (
    <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <span
        aria-hidden
        className={cn("size-1.5 rounded-full", isError ? "bg-bad" : isSuccess ? "bg-ok" : "bg-muted-foreground")}
      />
      {state}
    </span>
  )
}

export default function Operator() {
  const { data: metrics } = useMetrics()

  return (
    <div className="flex min-h-svh">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-12 shrink-0 items-center justify-between gap-3 border-b px-4">
          <div className="flex items-center gap-3">
            <h1 className="text-sm font-medium">Operator</h1>
            <ConnectionDot />
          </div>
          <div className="flex items-center gap-2">
            <ModeSwitch />
            <SimRunButton />
            <ResetButton />
          </div>
        </header>

        <main className="flex flex-col gap-3 p-3">
          <LiveCounters />
          <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_26rem]">
            <div className="flex min-w-0 flex-col gap-3">
              <ConfusionMatrixPanel metrics={metrics} />
              <div className="grid gap-3 lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
                <HeroStat />
                <OutcomeChart />
              </div>
              <div className="grid gap-3 lg:grid-cols-2">
                <HumanCost />
                <InvariantPanel />
              </div>
              <div className="grid gap-3 lg:grid-cols-2">
                <ModeCompare />
                <AttackLevelChart />
              </div>
              <TrustHistogram />
            </div>
            <div className="flex min-w-0 flex-col gap-3">
              <LedgerTable />
              <SimPanel />
              <ConfigEditor />
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}

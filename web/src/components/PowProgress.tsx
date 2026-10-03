/**
 * File: web/src/components/PowProgress.tsx
 * Purpose: Progress bar for the proof-of-work solve. Progress is an estimate
 *          (attempts / expected attempts) since PoW has no fixed end.
 */
import { Progress } from "@/components/ui/progress"
import type { PowState } from "@/hooks/useSession"

export function PowProgress({ pow, submitting }: { pow: PowState | null; submitting: boolean }) {
  // Asymptotic curve so the bar keeps moving but never claims 100% before the answer is found.
  const est = pow ? 1 - Math.exp(-pow.attempts / Math.max(1, pow.expected)) : 0
  const value = submitting ? 100 : Math.min(95, Math.round(est * 100))

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between text-sm">
        <span>{submitting ? "Checking your answer" : "Verifying you're human"}</span>
        <span className="num text-xs text-muted-foreground">{value}%</span>
      </div>
      <Progress value={value} aria-label="Verification progress" />
      <p className="text-xs text-muted-foreground">Keep this tab open. This usually takes a few seconds.</p>
    </div>
  )
}

/**
 * File: web/src/components/TickNumber.tsx
 * Purpose: Counter that tweens from its previous value to the new one.
 *          The only numeric motion allowed by the design rules.
 */
import { useEffect, useRef, useState } from "react"

export function TickNumber({
  value,
  format,
  className,
  duration = 400,
}: {
  value: number
  format: (n: number) => string
  className?: string
  duration?: number
}) {
  const [shown, setShown] = useState(value)
  const fromRef = useRef(value)
  const shownRef = useRef(value)

  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      shownRef.current = value
      setShown(value)
      return
    }
    fromRef.current = shownRef.current
    const start = performance.now()
    let raf = 0
    const step = (now: number) => {
      const t = Math.min(1, (now - start) / duration)
      const eased = 1 - (1 - t) ** 3
      const v = fromRef.current + (value - fromRef.current) * eased
      shownRef.current = v
      setShown(v)
      if (t < 1) raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [value, duration])

  return <span className={className}>{format(shown)}</span>
}

/**
 * File: web/src/components/ConfusionMatrixPanel.tsx
 * Purpose: Display Confusion Matrix (TP, FP, FN, TN), False Rejection Breakdown,
 *          and Precision / Recall / Human False Rejection Rate.
 */

import React from "react"
import { Metrics } from "@/api/client"
import { Panel } from "@/components/Panel"

interface Props {
  metrics?: Metrics | null
}

export const ConfusionMatrixPanel: React.FC<Props> = ({ metrics }) => {
  const tp = metrics?.tp ?? 0
  const fp = metrics?.fp ?? 0
  const fn = metrics?.fn ?? 0
  const tn = metrics?.tn ?? 0

  const total = tp + fp + fn + tn
  const pct = (val: number) => (total > 0 ? ((val / total) * 100).toFixed(1) + "%" : "0%")

  const precision = (metrics?.precision ?? (tp + fp > 0 ? tp / (tp + fp) : 0.0)) * 100
  const recall = (metrics?.recall ?? (tp + fn > 0 ? tp / (tp + fn) : 0.0)) * 100
  const fpRate = (metrics?.human_false_rejection_rate ?? (fp + tn > 0 ? fp / (fp + tn) : 0.0)) * 100

  const byReason = metrics?.by_reason ?? {}
  const byProfile = metrics?.by_profile ?? {}

  return (
    <Panel title="Evaluation & Confusion Matrix">
      <div className="space-y-4 text-xs">
        {/* Confusion Matrix 2x2 */}
        <div className="grid grid-cols-2 gap-2 text-center font-mono">
          <div className="bg-emerald-950/40 border border-emerald-800/60 p-3 rounded">
            <div className="text-emerald-400 font-bold text-sm">True Positives (TP)</div>
            <div className="text-xl font-bold text-emerald-200">{tp}</div>
            <div className="text-emerald-500/80 text-[10px]">Bots Blocked ({pct(tp)})</div>
          </div>
          <div className="bg-amber-950/40 border border-amber-800/60 p-3 rounded">
            <div className="text-amber-400 font-bold text-sm">False Negatives (FN)</div>
            <div className="text-xl font-bold text-amber-200">{fn}</div>
            <div className="text-amber-500/80 text-[10px]">Bots Passed ({pct(fn)})</div>
          </div>
          <div className="bg-rose-950/40 border border-rose-800/60 p-3 rounded">
            <div className="text-rose-400 font-bold text-sm">False Positives (FP)</div>
            <div className="text-xl font-bold text-rose-200">{fp}</div>
            <div className="text-rose-500/80 text-[10px]">Humans Wrongly Rejected ({pct(fp)})</div>
          </div>
          <div className="bg-blue-950/40 border border-blue-800/60 p-3 rounded">
            <div className="text-blue-400 font-bold text-sm">True Negatives (TN)</div>
            <div className="text-xl font-bold text-blue-200">{tn}</div>
            <div className="text-blue-500/80 text-[10px]">Humans Accepted ({pct(tn)})</div>
          </div>
        </div>

        {/* Headline Rates */}
        <div className="grid grid-cols-3 gap-2 text-center pt-1 border-t border-slate-800">
          <div>
            <div className="text-slate-400 text-[10px] uppercase">Precision</div>
            <div className="font-mono font-bold text-sm text-emerald-400">{precision.toFixed(1)}%</div>
          </div>
          <div>
            <div className="text-slate-400 text-[10px] uppercase">Recall</div>
            <div className="font-mono font-bold text-sm text-cyan-400">{recall.toFixed(1)}%</div>
          </div>
          <div>
            <div className="text-slate-400 text-[10px] uppercase">Human FP Rate</div>
            <div className="font-mono font-bold text-sm text-rose-400">{fpRate.toFixed(1)}%</div>
          </div>
        </div>

        {/* Human Rejections Breakdown */}
        {Object.keys(byReason).length > 0 && (
          <div className="pt-2 border-t border-slate-800">
            <div className="font-semibold text-slate-300 mb-1">Human Rejections by Reason Code:</div>
            <div className="space-y-1">
              {Object.entries(byReason).map(([reason, cnt]) => (
                <div key={reason} className="flex justify-between items-center bg-slate-900/60 px-2 py-1 rounded">
                  <span className="font-mono text-rose-300">{reason}</span>
                  <span className="font-mono font-bold text-rose-400">{cnt}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Human Rejections by Profile */}
        {Object.keys(byProfile).length > 0 && (
          <div className="pt-2 border-t border-slate-800">
            <div className="font-semibold text-slate-300 mb-1">Human Rejections by Profile:</div>
            <div className="space-y-1">
              {Object.entries(byProfile).map(([profile, cnt]) => (
                <div key={profile} className="flex justify-between items-center bg-slate-900/60 px-2 py-1 rounded">
                  <span className="font-mono text-amber-300">{profile}</span>
                  <span className="font-mono font-bold text-amber-400">{cnt}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Panel>
  )
}

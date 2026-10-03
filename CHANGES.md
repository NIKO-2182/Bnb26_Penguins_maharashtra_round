# FairDrop Web: Change Log

This file lists everything added or changed to build the FairDrop frontend (`web/`) and get it running in the v0 preview. Use it to check the code against the spec.

---

## 1. Project layout

```
/                      repo root (thin wrapper, runs the web app)
├── package.json       rewritten: scripts delegate to web/
├── public/
└── web/               the Vite + React SPA (all real code lives here)
```

### Root-level changes

| Change | Why |
| --- | --- |
| **Deleted** `next.config.mjs`, `next-env.d.ts`, `app/page.tsx`, `app/layout.tsx`, `app/globals.css`, `postcss.config.mjs`, `components.json`, `components/ui/button.tsx`, `lib/utils.ts`, `tsconfig.json` | Leftover Next.js scaffold. It was serving a placeholder on port 3000, so the preview never showed the real app. |
| **Rewrote** `package.json` | No Next.js dependency anymore. Scripts now forward to `web/`. |

Root `package.json` scripts:

```json
"postinstall": "pnpm --dir web install",
"dev":   "pnpm --dir web dev",
"build": "pnpm --dir web build",
"start": "pnpm --dir web preview --host 0.0.0.0 --port 3000"
```

---

## 2. Stack (`web/package.json`)

| Area | Choice |
| --- | --- |
| Build / dev server | Vite 8, `@vitejs/plugin-react` |
| UI | React 19, Tailwind CSS v4 (`@tailwindcss/vite`), shadcn components (Base UI primitives) |
| Routing | `react-router-dom` v7 |
| Server state | `@tanstack/react-query` v5 (polling) |
| Client state | `zustand` v5 (admin controls) |
| Charts | `recharts` v3 |
| Icons / fonts | `lucide-react`, Inter Variable + JetBrains Mono Variable (`@fontsource-variable`) |
| Types | TypeScript 5.7 (`tsc --noEmit` runs before every build) |

Scripts: `dev` (Vite on `0.0.0.0:3000`), `build` (typecheck, then vite build), `preview`, `typecheck`.

---

## 3. Config and infrastructure files

| File | Purpose |
| --- | --- |
| `web/vite.config.ts` | `@` alias to `src/`. Dev server on `:3000`, `allowedHosts: true`. **Dev proxy:** `/api` → `FAIRDROP_API_URL` (default `http://localhost:8080`) and `/sim` → `FAIRDROP_SIM_URL` (default `http://localhost:8090`), with the prefix stripped. Workers built as ES modules. |
| `web/tsconfig.json` | Strict TS, `@/*` path alias. |
| `web/index.html` | SPA entry, loads `src/main.tsx`. |
| `web/components.json` | shadcn config. |
| `web/public/icon.svg` | Favicon. |
| `web/Dockerfile` | Two stages: `node:22-alpine` builds with pnpm, then `nginx:1.27-alpine` serves `dist/` on `:3000`. |
| `web/nginx.conf` | Serves the SPA (`try_files ... /index.html`). Proxies `/api/` → `http://api:8080/` and `/sim/` → `http://sim:8090/` (prefix stripped, same as the dev proxy). Long-term cache on `/assets/`, `no-cache` on HTML. Security headers: `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `X-Frame-Options: SAMEORIGIN`. |

> Check: the docker-compose service names must be `api` (port 8080) and `sim` (port 8090) for the nginx proxy to resolve.

---

## 4. Routes (`web/src/App.tsx`)

| Path | Page | Notes |
| --- | --- | --- |
| `/` | `pages/UserView.tsx` | Public "join the drop" flow. Loaded eagerly. |
| `/operator` | `pages/Operator.tsx` | Operator dashboard. **Lazy-loaded** so the user bundle doesn't include Recharts. |
| `*` | redirect to `/` | |

---

## 5. API layer

### `web/src/api/client.ts`: core API (base `VITE_API_BASE`, default `/api`)

| Function | Method / path |
| --- | --- |
| `getHealth` | `GET /health` |
| `getMetrics` | `GET /metrics` |
| `getLedger(limit=50)` | `GET /ledger?limit=N` |
| `getResults` | `GET /results` |
| `join` | `POST /join` → PoW challenge + token |
| `verify(token, nonce)` | `POST /verify` |
| `getSession(token)` | `GET /session?token=...` (also sends the auth header) |
| `claim(token)` | `POST /claim` |
| `setMode(mode)` | `POST /admin/mode` `{ mode }` (`"fair"` or `"fcfs"`) |
| `setConfig(json)` | `POST /admin/config` `{ config_json }` |
| `resetAll` | `POST /admin/reset` |

Also exports: the `ApiError` class, the generic `request()` helper, and the types `Mode`, `UserType`, `Metrics`, `LedgerEvent`, `ResultRun`, `PowChallenge`, `JoinResponse`, `SessionRaw`.

### `web/src/api/sim.ts`: simulator (base `VITE_SIM_BASE`, default `/sim`)

| Function | Method / path |
| --- | --- |
| `runSim(config)` | `POST /run` |
| `stopSim` | `POST /stop` |
| `getSimStatus` | `GET /status` |

`BOT_PROFILES = ["naive", "solver", "distributed", "retry"]`, plus the `SimConfig` and `SimStatus` types.

> Check: compare these paths and payload field names with the backend handlers.

---

## 6. Hooks, store, and lib

| File | What it does |
| --- | --- |
| `hooks/useMetrics.ts` | `useMetrics()`, `useResults()`, `useSimStatus()`: React Query polling hooks. |
| `hooks/useLedger.ts` | `useLedger(limit)`: polls the ledger and gives each event a stable key (`KeyedEvent`), so only new rows animate in. |
| `hooks/useSession.ts` | User flow state machine (`Phase`): join, solve PoW in a worker, verify, poll session, claim. Exposes `PowState` (attempts, hash rate, elapsed time). |
| `store/adminStore.ts` | Zustand store for operator/admin UI state. |
| `lib/pow.worker.ts` | Web Worker. Finds a decimal nonce where `SHA-256(challenge + nonce)` has at least `difficulty` leading zero bits. Uses a **synchronous** SHA-256 so the hot loop doesn't pay `crypto.subtle` promise overhead. Messages out: `{type:"progress", attempts, hashRate}` and `{type:"done", nonce, hash, attempts, ms}`. |
| `lib/ledger.ts` | `eventTone()` (ok/warn/bad/neutral), tone-to-class maps, `label()` to make reason codes readable, `deriveResult()`. |
| `lib/results.ts` | `latestPerMode()` and `byIntensity()` for the comparison charts. |
| `lib/chart.ts` | Shared Recharts axis, grid, and tooltip props. |
| `lib/utils.ts` | `cn`, `fmtPct`, `fmtInt`, `fmtTime`, `shortId`. |

> Check: the leading-zero-**bits** rule in `pow.worker.ts` must match how the backend verifies difficulty (bits, not hex characters).

---

## 7. Components

### Operator dashboard (`/operator`)

| Component | Purpose |
| --- | --- |
| `Panel.tsx` | Shared card/panel shell. |
| `TickNumber.tsx` | Number that animates when its value changes. |
| `HeroStat.tsx` | Headline fairness stat. |
| `LiveCounters.tsx` | Live counters from `/metrics`. |
| `HumanCost.tsx` | What fairness costs real humans (e.g. friction/latency). |
| `InvariantPanel.tsx` | Pass/fail invariant checks (ok/warn/bad tones). |
| `OutcomeChart.tsx` | Human vs. bot outcomes. |
| `ModeCompare.tsx` | Fair vs. FCFS comparison from `/results`. |
| `AttackLevelChart.tsx` | Outcomes by attack intensity. |
| `TrustHistogram.tsx` | Distribution of trust scores. |
| `LedgerTable.tsx` | Live ledger. New rows animate on mount (stable keys). |
| `LedgerDrawer.tsx` | Side sheet with ledger detail, filterable by **reason** and **user type**. Renamed from `EventDrawer.tsx` to match the spec. |
| `SimControls.tsx` | Start/stop the simulator, choose bot profiles and settings. |
| `AdminControls.tsx` | Mode switch, config editor, reset (confirmed through an alert dialog). |

### User view (`/`)

| Component | Purpose |
| --- | --- |
| `PowProgress.tsx` | Proof-of-work progress (attempts, hash rate). |
| `StatusCard.tsx` | Current queue/session/claim status. |

### shadcn UI (`components/ui/`)

`alert-dialog`, `badge`, `button`, `label`, `progress`, `select`, `separator`, `sheet`, `skeleton`, `slider`, `switch`, `textarea`, `tooltip`

---

## 8. Styling (`web/src/index.css`)

- Tailwind v4 with `@theme` design tokens and a dark operator look.
- Semantic status tokens `ok`, `warn`, `bad`, plus a tabular-number `num` utility for numbers.
- Fonts: Inter (UI) and JetBrains Mono (numbers, hashes, IDs).

---

## 9. Preview fix

**Problem:** The preview showed nothing useful, because port 3000 was serving the leftover Next.js scaffold instead of the Vite app.

**Fix:**
1. Deleted the Next.js scaffold files (section 1).
2. Changed the root `package.json` so `pnpm dev` runs `vite --host 0.0.0.0 --port 3000` inside `web/`.
3. Set `allowedHosts: true` in `vite.config.ts` so Vite accepts the preview's proxied host.

---

## 10. Verification done

- `tsc --noEmit`: passed.
- `vite build`: passed. Operator is a separate lazy chunk.
- Browser: `/` renders the "Join the drop" view on mobile and desktop. `/operator` renders with no console errors.

### Known caveat

The Go API (`:8080`) and simulator (`:8090`) **don't run in the v0 sandbox**. In the preview, data panels show empty or error states and the join flow can't finish. To test end to end, run the backend services (docker-compose), or point the dev proxy at them:

```bash
FAIRDROP_API_URL=http://<api-host>:8080 FAIRDROP_SIM_URL=http://<sim-host>:8090 pnpm dev
```

---

## 11. Checklist

- [ ] API paths and payload fields (section 5) match the backend
- [ ] PoW difficulty is in leading zero **bits** on both sides
- [ ] Compose service names are `api:8080` and `sim:8090` (nginx)
- [ ] Ledger reason codes and user types match what `LedgerDrawer` filters on
- [ ] Bot profiles `naive / solver / distributed / retry` match the simulator
- [ ] Mode values `fair` / `fcfs` match `/admin/mode`

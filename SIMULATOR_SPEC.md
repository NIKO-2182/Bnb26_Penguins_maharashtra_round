# SIMULATOR_SPEC.md: Fair Drop Stress Test

## 1. Goal
Measure, with reproducible numbers, how Fair Drop handles bot attacks while keeping real humans (including frustrated fast-clickers and shared-IP users) from being wrongly rejected. Compare against a first-come-first-served (FCFS) baseline.

## 2. Principles
- **Full flow only.** Every virtual user runs `POST /join -> solve PoW -> POST /verify -> GET /session (poll) -> POST /claim`. The simulator never calls `/claim` directly to skip the abuse layer.
- **No ground truth in decisions.** The simulator sends `user_type` and `human_profile` in headers accepted only when `SIM_MODE=true`. The API writes them to the ledger for evaluation. No decision code (rate limit, trust, draw, claim) may read them.
- **Trust is computed, never assigned.** The API derives trust from observed behaviour (timing, PoW, retries, IP, session continuity).
- **Reproducible.** Every run takes a `seed`, saved in the report.
- **Containerized.** Runs as the `sim` service inside the compose network, never from the host.

## 3. Architecture
- Go service in `sim/`, one goroutine per virtual user
- Control API on `:8090`: `POST /run` (scenario config), `POST /stop`, `GET /status`
- Scenario presets in `sim/internal/scenarios/*.yaml`
- Reports written to `results/<scenario>_<mode>_<seed>.json`
- Each virtual user sends `X-Forwarded-For` from its assigned IP (trusted by the API only in `SIM_MODE`)

## 4. Human profiles
| Profile | Behaviour |
|---|---|
| `human_normal` | Variable think time, one solve, one claim attempt, occasional single retry |
| `human_slow` | High latency, mobile-like jitter, slow PoW solve |
| `human_frustrated` | Rapid clicks and refreshes in short bursts, then pauses; retries from the same session token and IP; irregular gaps |
| `human_shared_ip` | Many humans behind a few IPs (e.g. 200 humans on 5 IPs), normal behaviour otherwise |

## 5. Bot profiles
| Profile | Behaviour |
|---|---|
| `bot_naive` | Single IP flood, no PoW solving |
| `bot_solver` | Solves PoW at machine speed, uniform timing |
| `bot_distributed` | Thousands of IPs, low rate per IP, solves PoW |
| `bot_retry` | Aggressive retries, reconnect storms, token churn |

## 6. Scenarios (YAML)
Each scenario sets: `seats`, `window_seconds`, `humans` (count per profile), `bots` (count per profile), `ip_pools`, `arrival_pattern`, `seed`, `mode` (`fair` or `fcfs`).

- **baseline_no_bots:** humans only. Confirms normal operation and the human win rate.
- **flash_sale:** 500 seats, 2,500 humans, 25,000 bots (10x). Everyone starts together.
- **late_arrivals:** a share of humans join in the last seconds of the window. Tests that arrival time does not change odds.
- **shared_ip:** 200 humans behind 5 IPs plus a bot attack.
- **sweep:** same setup at 1x, 5x, 10x and 50x bots, for both `fcfs` and `fair`.
- **unseen_attack:** one bot profile (e.g. `bot_distributed`) is excluded from trust-weight tuning and reported separately.

Scarcity rule: requesters must always far exceed seats, otherwise the result is meaningless.

## 7. Report (JSON, per run)
- `scenario`, `mode`, `seed`, `seats`, `humans`, `bots`, `duration`
- **Confusion matrix:** `tp` (bots blocked), `fn` (bots let through), `fp` (humans wrongly rejected), `tn` (humans accepted), plus precision and recall
- **Seats won** by user type, and `bot_share_of_seats`
- **Human win rate** and expected rate (`seats / humans_in_pool`)
- **Human false rejections** by reason code and by human profile
- **By bot profile:** requests, accepted, rejected, 429s
- **Latency:** p50, p95, p99
- **Invariants:** `oversell_count`, `duplicate_count`
- **Cross-check:** `sim_side_totals` vs `ledger_totals`; flag any mismatch

## 8. Success criteria
| Metric | Target |
|---|---|
| Oversell and duplicate allocations | 0 (hard fail otherwise) |
| Bot share of seats (fair) | Far below FCFS at the same attack level |
| Human false-rejection rate | Within `FP_BUDGET` (e.g. <= 2%) |
| `human_frustrated` false-rejection rate | Reported separately and low |
| Human win rate | Close to `seats / humans` |
| Late-window human win rate | Statistically similar to early-window humans |
| p95 latency | Reported at each attack level |
| Sim vs ledger totals | Equal |

## 9. Tuning discipline
- Tune trust weights and thresholds on one set of seeds
- Report final numbers on different seeds
- Report `unseen_attack` results separately, with no tuning on that profile
- Do not tune on the scenario used for the headline result

## 10. Files (`sim/`)
Each file starts with the standard header comment block from `AGENTS.md`.
```
Dockerfile, go.mod
main.go                           starts control server on :8090
internal/control/server.go        /run, /stop, /status
internal/runner/runner.go         spawns virtual users, collects outcomes
internal/runner/client.go         full-flow HTTP client (join, solve, verify, poll, claim)
internal/runner/pow.go            PoW solver used by solver-type users
internal/runner/ipgen.go          IP pools for single, shared and distributed cases
internal/profiles/human_*.go      normal, slow, frustrated, shared_ip
internal/profiles/bot_*.go        naive, solver, distributed, retry
internal/scenarios/*.yaml         scenario presets
internal/report/report.go         builds report, confusion matrix, cross-check
internal/report/ledger_pull.go    reads the API ledger for the cross-check
```

## 11. Roadmap
- [ ] 1. Full-flow client (join, PoW, verify, poll, claim)
- [ ] 2. Human and bot profiles, IP generation, `user_type` headers
- [ ] 3. Scenario loader and control API
- [ ] 4. Report generator with confusion matrix and sim-vs-ledger cross-check
- [ ] 5. Runs: baseline_no_bots, then FCFS with bots, then Fair with bots
- [ ] 6. Sweep (1x, 5x, 10x, 50x) with tune/report seed split
- [ ] 7. Late-arrivals, shared-IP and unseen-attack runs

## 12. Backend requirements
- Accept `user_type` and `human_profile` headers only in `SIM_MODE`; store them in the ledger; never read them in decision code
- Ledger event includes `ip`, `subnet`, `trust_score`, `reason_code`
- Metrics expose confusion matrix counts, precision, recall, human false-rejection rate and invariant counts
- Cooldown handling for fast clickers (soft penalty, not a ban)
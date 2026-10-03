// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/AGENTS.md
// PURPOSE: Rules for the coding agent
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: coding agent
// RULES: build order, header standard, "run docker compose before claiming done", never add dependencies without asking
// DO NOT: change header standard

# Agent Rules

## Build Order
1. `docs/API_CONTRACT.md` and `api/internal/ledger/event.go`
2. Compose file, both Dockerfiles, and the health endpoint
3. `allocator/claim.lua`, `claim.go`, `claim_test.go` (must pass before moving on)
4. Session, token, join, session handlers
5. Ratelimit, PoW, verify, trust
6. Window scheduler and draw, `fcfs.go`
7. Ledger writer and schema
8. Sim profiles and runner
9. Metrics, invariants, WebSocket hub
10. Dashboard
11. `run_matrix.sh`, results, README

## Standards
- **Header Standard**: Every file must have the header format defined in the prompt.
- **Dependencies**: Never add new dependencies without asking.
- **Execution**: Finish and test each step before starting the next.
- **Docker**: Always run `docker compose up` before claiming a task is "done".
- **Invariants**: Never edit a file if the header says `DO NOT` for that specific change.

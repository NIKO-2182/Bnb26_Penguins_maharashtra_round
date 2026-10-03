# Project Progress: Fair Drop System

## Summary of Completed Tasks
- [x] **Infrastructure Foundation**:
    - Created `ledger` package with Redis persistence and `Event` schemas.
    - Created `metrics` package and integrated it with `ledger`.
    - Updated `httpx.Router` to inject `ledger` and `metrics` dependencies.
- [x] **Abuse Prevention & Monitoring**:
    - Implemented `RateLimiter` instrumentation for real-time RPS and 429 error tracking.
- [x] **Core Logic (Initial)**:
    - **Trust Scoring System**: Defined `TrustManager` to handle user trust scores.
    - **Atomic Claim Logic**: Created and updated the Lua-based claim script (`claim.lua`) to support trust-weighted random draws.
- [x] **Git & Versioning**: Initial commit established for the core skeleton.

## Remaining Backend Tasks (Sprint to 9:30 PM)
- [ ] **Trust & Claim Integration**:
    - Connect `TrustManager` to `ClaimHandler` to retrieve user scores.
    - Refactor `ClaimHandler.HandleClaim` to generate a random value and pass both `trustScore` and `random_val` to the allocator.
- [ ] **API Endpoint Wiring**:
    - Implement logic for `/metrics` to surface live pool data and error rates.
    - Implement logic for `/ledger` to expose the event stream.
- [ ] **Admin Controls**:
    - Implement a basic "Mode" toggle (Fair Drop ON/OFF) via configuration or simple flag.

## Technical Context
- **Database**: Redis (Primary for claims, sessions, and trust scores).
- **Atomic Operation**: `api/internal/allocator/claim.lua` handles the core "Compare-and-Swap" for seat allocation.
- **Metrics**: Tracked via `api/internal/metrics/service.go`.

## Resume Instructions
1. **Priority**: Connect the `TrustManager` to the `ClaimHandler`.
2. **Key Files to Edit**:
    - `api/internal/handlers/claim.go` (Connect to `TrustManager`)
    - `api/internal/allocator/claim.go` (Update `ClaimSeat` signature)
    - `api/internal/httpx/router.go` (Wire the new metrics/ledger handlers)
3. **Verification**: Ensure that a "Bot" (low trust score) has a mathematically lower probability of successful claim than a "Human" (high trust score) during a high-concurrency simulation.

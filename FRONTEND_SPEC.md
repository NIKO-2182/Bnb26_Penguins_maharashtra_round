# Frontend Specification: Fair Drop Dashboard

## Goal
Build a real-time monitoring and administration dashboard to visualize the "Fair Drop" system's efficacy and manage system states.

## Core Features
### 1. Real-time Metrics Dashboard
- **Live RPS Counter**: Track incoming requests per second.
- **429 Error Rate**: Visualize how many requests are being throttled by the RateLimiter.
- **Seat Pool Status**: A visual gauge of remaining seats (e.g., "450/1000 Seats Available").
- **Trust Distribution**: A chart showing the average trust score of active sessions.

### 2. The "Live Ledger" Feed
- A scrolling ticker of the latest events from the `ledger` service.
- **Color Coding**: 
    - Green: `seat_granted`
    - Yellow: `rejected_low_trust`
    - Red: `duplicate_claim` or `sold_out`
- **Detail View**: Clicking an event shows the user ID, trust score at time of claim, and the result.

### 3. Admin Control Panel
- **Fair Drop Mode Toggle**: Master switch to set mode via `POST /admin/mode` (`{ "mode": "fair|fcfs" }`).
- **Sensitivity & Config Adjustment**: Adjust system config JSON via `POST /admin/config` (`{ "config_json": "..." }`).
- **Manual Reset**: Emergency flush button invoking `POST /admin/reset` to reset seats, claimed sets, and metrics.

## Technical Requirements
- **Framework**: React / Vite / Next.js with Tailwind CSS.
- **Communication**: 
    - Use **`GET /ledger?limit=50`** polling/stream for the Live Ledger feed.
    - Use **Polling (every 1s)** on **`GET /metrics`** for the Metrics Dashboard.
- **State Management**: Simple React context or Zustand for handling global config state.

## Integration Points
- `GET /metrics`: Returns `{ "rps": 0, "429s": 0, "seats_left": 500, "pool_size": 0, "bot_share": 0.0 }`.
- `GET /ledger`: Returns recent audit events array `[ { "timestamp": "...", "user_id": "...", "event_type": "...", "reason_code": "..." } ]`.
- `POST /admin/mode`: Toggles execution mode (`{ "mode": "fair|fcfs" }`).
- `POST /admin/reset`: Resets seat pool and clears Redis metrics.
- `POST /admin/config`: Updates system parameters.
- `POST /join`, `POST /verify`, `GET /session`, `POST /claim`: Core user claim flow endpoints.

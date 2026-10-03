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
- **Fair Drop Toggle**: A master switch to enable/disable the trust-weighted lottery.
- **Sensitivity Slider**: Adjust the "strictness" of the trust score (how much weight the score has vs the random draw).
- **Manual Override**: A button to "Flush" the claimed set or "Reset" the seat pool in an emergency.

## Technical Requirements
- **Framework**: React / Next.js with Tailwind CSS.
- **Communication**: 
    - Use **SSE (Server-Sent Events)** for the Live Ledger feed.
    - Use **Polling (every 1s)** for the Metrics Dashboard.
- **State Management**: Simple React context or Zustand for handling global config state.

## Integration Points
- `GET /metrics`: To populate the dashboard charts.
- `GET /ledger`: To populate the live feed.
- `POST /admin/config`: To update the system's "Fair Drop" sensitivity.

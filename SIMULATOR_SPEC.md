# Simulator Specification: Fair Drop Stress Test

## Goal
Build a standalone simulation tool to validate the "Fair Drop" system's ability to mitigate bot attacks while maintaining a high success rate for human users.

## Simulator Architecture
- **Architecture**: A separate Go or Python application that generates concurrent HTTP requests against the `/claim` endpoint.
- **Execution**: The simulator should be able to run in a containerized environment (Docker) to simulate high-concurrency networking.

## Simulation Profiles
### 1. "The Bot Swarm" (Attack Profile)
- **Identity**: 500+ unique IDs with "Low Trust" scores.
- **Behavior**: Constant, high-frequency requests (as fast as the rate limiter allows).
- **Goal**: Exhaust the seat pool.

### 2. "The Human Wave" (Legitimate Profile)
- **Identity**: 100 unique IDs with "High Trust" scores (pre-verified/high POW).
- **Behavior**: Variable delays (simulating human thinking/clicking times).
- **Goal**: Successfully claim seats despite the bot swarm.

### 3. "The Flash Sale" (Scenario)
- **Setup**: 1000 seats, 1000 bots, 100 humans.
- **Action**: Triggered simultaneously.
- **Success Metric**: At least 80% of the human users should successfully claim a seat, while less than 5% of the bots should succeed.

## Verification Metrics (Reporting)
After each run, the simulator must generate a JSON report containing:
- **Total Request Count** (Bot vs Human)
- **Success Rate** (Bot vs Human)
- **Average Latency** (P50, P95, P99)
- **Fairness Score**: `(Human_Success_Rate / Bot_Success_Rate) * 100`
- **Rate Limiter Hits**: Total number of 429 errors triggered.

## Development Roadmap
- [ ] **Step 1**: Create `sim/main.go` to handle basic HTTP client logic.
- [ ] **Step 2**: Implement the "Profile" logic (Randomized delays and header injection).
- [ ] **Step 3**: Create a "Report Generator" that parses the `ledger` data to calculate the final Fairness Score.
- [ ] **Step 4**: Run a "Baseline Test" with 0 bots to ensure standard functionality.
- [ ] **Step 5**: Run the "Flash Sale" test and iterate on the Trust Score weights.

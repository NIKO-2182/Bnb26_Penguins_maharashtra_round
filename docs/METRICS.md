// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/docs/METRICS.md
// PURPOSE: Metric definitions
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: Developers, Dashboard
// RULES: N/A
// DO NOT: N/A

# Metrics

## Live Counters
- `RPS`: Requests per second
- `429s`: Rate limit triggers per second
- `Seats Left`: Total seats - Granted seats
- `Pool Size`: Current number of verified users in the pool

## Fairness Metrics
- `Bot Share`: (Seats granted to bots) / (Total seats granted)
- `Human Win Rate`: (Seats granted to humans) / (Total humans in pool)
- `Expected Win Rate`: (Total humans in pool) / (Total pool size)
- `Human False Rejection Rate`: (Humans rejected by trust score) / (Total humans)
- `Allocation Spread`: Variance of distribution across different user types

## Latency
- `p50/p95/p99`: Request latency histograms

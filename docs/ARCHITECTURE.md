// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/docs/ARCHITECTURE.md
// PURPOSE: Architecture diagram and component responsibilities
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: Developers, Users
// RULES: N/A
// DO NOT: N/A

# Architecture

(Flow described in the prompt)

## Components
- **API**: Handles joins, verification, claims, and state management.
- **Redis**: Stores sessions, rate limits, and the selection pool.
- **Postgres**: Persistent ledger for all events.
- **Window Scheduler**: Manages the "Join Window" and triggers the draw.
- **Allocator**: Performs trust-weighted random selection.
- **Sim**: Synthetic bot/human traffic generator.
- **Web**: Operator dashboard and User frontend.

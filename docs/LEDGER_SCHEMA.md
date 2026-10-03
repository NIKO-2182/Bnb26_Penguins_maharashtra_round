// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/docs/LEDGER_SCHEMA.md
// PURPOSE: Database schema for the ledger
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: Developers
// RULES: N/A
// DO NOT: N/A

# Ledger Schema

## Table: `ledger`
| Column | Type | Description |
|---|---|---|
| id | bigint | Primary Key |
| timestamp | timestamptz | Time of event |
| user_id | uuid | Unique identifier for user |
| event_type | varchar | Action performed |
| reason_code | varchar | Specific reason for outcome |
| metadata | jsonb | Additional context |

## Reason Codes
- `accepted_pool`
- `selected`
- `seat_granted`
- `rejected_ratelimit_ip`
- `rejected_ratelimit_token`
- `rejected_pow_invalid`
- `rejected_pow_too_fast`
- `rejected_low_trust`
- `not_selected_draw`
- `sold_out`
- `duplicate_claim`

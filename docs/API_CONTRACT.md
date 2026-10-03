// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/docs/API_CONTRACT.md
// PURPOSE: API contract for the FairDrop service
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: Frontend, Sim, Agents
// RULES: Source of truth for all endpoints
// DO NOT: N/A

# API Contract

## POST /join
- **Request**: `{ "username": "string" }`
- **Response**: `200 OK { "session_token": "string", "pow_challenge": "string" }`
- **Reason Codes**: `none`

## POST /verify
- **Request**: `{ "session_token": "string", "solution": "string" }`
- **Response**: `200 OK { "status": "verified" }`
- **Reason Codes**: `rejected_pow_invalid`, `rejected_pow_too_fast`

## GET /session
- **Request**: `{ "session_token": "string" }`
- **Response**: `200 OK { "state": "new|challenged|verified|pooled|selected|claimed|not_selected|rejected" }`

## POST /claim
- **Request**: `{ "claim_token": "string" }`
- **Response**: `200 OK { "status": "granted|duplicate|sold_out" }`

## GET /metrics
- **Response**: `200 OK { "rps": 0, "seats_left": 0, "pool_size": 0, "bot_share": 0.0 }`

## GET /ledger
- **Query Params**: `?limit=50&offset=0`
- **Response**: `200 OK [ { "id": 1, "event": "...", "reason_code": "..." } ]`

## POST /admin/mode
- **Request**: `{ "mode": "fair|fcfs" }`
- **Response**: `200 OK { "status": "success" }`

## POST /admin/reset
- **Response**: `200 OK { "status": "success" }`

## POST /admin/config
- **Request**: `{ "config_json": "{...}" }`
- **Response**: `200 OK { "status": "success" }`

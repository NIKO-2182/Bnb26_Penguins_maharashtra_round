-- FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/allocator/claim.lua
-- PURPOSE: Atomic claim logic for seats with trust-weighted lottery
-- INPUTS / OUTPUTS: N/A
-- DEPENDS ON: N/A
-- USED BY: claim.go
-- RULES: Must ensure no oversell and no duplicate claims
-- DO NOT: N/A

local seats_left = redis.call('get', 'seats:left')
if not seats_left or tonumber(seats_left) <= 0 then
    return "sold_out"
end

local claim_token = args[1]
local random_val = tonumber(args[2])
local trust_score = tonumber(args[3])

if redis.call('sismember', 'seats:claimed', claim_token) == 1 then
    return "duplicate_claim"
end

-- Trust-weighted random draw
-- If the random value (0-1) is greater than the trust score, the claim is rejected.
if random_val > trust_score then
    return "rejected_low_trust"
end

redis.call('decr', 'seats:left')
redis.call('sadd', 'seats:claimed', claim_token)

return "seat_granted"

// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/allocator/claim.go
// PURPOSE: Executes the claim logic via Lua with auto-initializing seats
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go
// USED BY: handlers/claim.go
// RULES: Must call Lua script for atomicity
// DO NOT: N/A
package allocator

import (
	"context"
	"fmt"
	"time"

	"fairdrop/api/internal/store"
)

// MaxDrawAttempts bounds how many times one session may enter the lottery.
// Beyond this the session is refused outright rather than being allowed a
// vanishing chance, which keeps a retry flood from burning Redis round-trips.
const MaxDrawAttempts = 8

// MaxIPDrawAttempts bounds total draw attempts from one source address. This
// is the defence that survives session churn: a flood that mints a new token
// per request still runs out of budget at this address. Set high enough to
// accommodate the shared-IP human scenario (200 humans / 5 IPs).
const MaxIPDrawAttempts = 120

// MinTrustToDraw is the trust floor for entering the lottery at all. Below it
// a session is refused rather than given a small chance, so a low-trust flood
// cannot win seats by brute force over many attempts.
const MinTrustToDraw = 0.30

// FreeAttempts is how many draws a session gets at UNDILUTED trust. Retry
// decay used to apply from attempt 1 and measurably punished legitimate slow
// humans; volume is already capped by the per-IP ceiling and by one-win-per-
// token, so retries only need bounding, not taxing.
const FreeAttempts = 4

// MinEffectiveTrust is the floor a decayed draw probability falls to. Past the
// free allowance a spammer keeps a token chance instead of being hard-blocked,
// which matters because hard blocks are what produced the false rejections.
const MinEffectiveTrust = 0.15

type ClaimResult struct {
	Status string
}

// claimSeatCache counts how many draw attempts this session has made.
func claimSeatCache(ctx context.Context, rdb *store.RedisClient, claimToken string, randomVal float64, trustScore float64, ip string, fairMode bool) (*ClaimResult, int, error) {
	// Count this attempt BEFORE drawing so the divisor includes it.
	var attempt int64
	if claimToken != "" {
		attempt, _ = rdb.Client.Incr(ctx, "attempts:"+claimToken).Result()
		rdb.Client.Expire(ctx, "attempts:"+claimToken, 1*time.Hour)
	}
	if attempt < 1 {
		attempt = 1
	}

	fair := 0
	if fairMode {
		fair = 1
	}

	res, err := rdb.Client.Eval(ctx, `
		local seats_left = redis.call('get', 'seats:left')
		if not seats_left then
			redis.call('set', 'seats:left', '500')
			seats_left = '500'
		end
		if tonumber(seats_left) <= 0 then
			return "sold_out"
		end

		local claim_token = ARGV[1]
		local random_val = tonumber(ARGV[2])
		local trust_score = tonumber(ARGV[3])
		local attempt = tonumber(ARGV[4])
		local max_attempts = tonumber(ARGV[5])
		local ip = ARGV[6]
		local max_ip_attempts = tonumber(ARGV[7])
		local min_trust = tonumber(ARGV[8])
		local fair = tonumber(ARGV[9])
		local free_attempts = tonumber(ARGV[10])
		local min_effective = tonumber(ARGV[11])

		if redis.call('sismember', 'seats:claimed', claim_token) == 1 then
			return "duplicate_claim"
		end

		-- All anti-abuse ceilings below are bypassed in fcfs mode, which is the
		-- control arm and must remain a bare first-come-first-served baseline.
		if fair == 0 then
			if random_val > trust_score then
				return "rejected_low_trust"
			end
			redis.call('decr', 'seats:left')
			redis.call('sadd', 'seats:claimed', claim_token)
			return "seat_granted"
		end

		-- Per-IP ceiling. Retry decay alone can be defeated by minting fresh
		-- sessions from the same address: each new token starts its attempt
		-- count at 1 and gets an undiluted draw. This bounds how many seats one
		-- source address may draw for in total, which is what actually stops a
		-- single-IP flood (bot_naive) regardless of session churn.
		if ip ~= '' then
			local ip_attempts = tonumber(redis.call('incr', 'ipattempts:' .. ip))
			redis.call('expire', 'ipattempts:' .. ip, 3600)
			if ip_attempts > max_ip_attempts then
				return "rejected_attempt_limit"
			end
		end

		-- A session below the trust floor does not draw at all. Without this,
		-- low-trust clients keep rolling the dice: at trust 0.05 a bot still wins
		-- one seat every ~20 attempts, so a flood eventually lands seats.
		if trust_score < min_trust then
			return "rejected_low_trust"
		end

		-- Retry policy: a small FREE allowance, then a floor rather than linear decay.
		--
		-- An earlier version divided trust by the attempt number (trust/n).
		-- That looks principled but measured badly: it punished exactly the
		-- people the project exists to protect. Slow humans legitimately retry
		-- after losing a draw, and at n=4 effective trust had fallen to ~0.21,
		-- so 608 legitimate human requests were rejected for the crime of
		-- trying again.
		--
		-- Volume is already neutralised by two stronger guarantees: a token can
		-- win at most once (SADD seats:claimed) and a source address has a hard
		-- ceiling on draws. So retries here need only be bounded, not taxed.
		-- The first FreeAttempts are undiluted, giving an honest user a fair
		-- shot at several draws; beyond that the chance decays to a floor so a
		-- spammer gains nothing while still not being hard-blocked.
		local effective = trust_score
		if attempt > free_attempts then
			local over = attempt - free_attempts
			local decayed = trust_score / over
			effective = math.max(decayed, min_effective)
			if effective > trust_score then
				effective = trust_score
			end
		end

		-- Past the attempt ceiling, stop drawing entirely rather than letting a
		-- spammer retry forever for a vanishing chance.
		if attempt > max_attempts then
			return "rejected_attempt_limit"
		end

		if random_val > effective then
			return "rejected_low_trust"
		end

		redis.call('decr', 'seats:left')
		redis.call('sadd', 'seats:claimed', claim_token)

		return "seat_granted"
	`, []string{}, claimToken, fmt.Sprintf("%.6f", randomVal), fmt.Sprintf("%.6f", trustScore), attempt, MaxDrawAttempts, ip, MaxIPDrawAttempts, MinTrustToDraw, fair, FreeAttempts, MinEffectiveTrust).Result()

	if err != nil {
		return nil, int(attempt), fmt.Errorf("lua error: %w", err)
	}

	statusStr, ok := res.(string)
	if !ok {
		return nil, int(attempt), fmt.Errorf("unexpected lua result type")
	}

	return &ClaimResult{Status: statusStr}, int(attempt), nil
}

// ClaimSeat draws for a seat. The returned int is the 1-based attempt number
// for this session, which the caller records in the ledger for auditability.
func ClaimSeat(ctx context.Context, rdb *store.RedisClient, claimToken string, trustScore float64, randomVal float64, ip string, fairMode bool) (*ClaimResult, int, error) {
	return claimSeatCache(ctx, rdb, claimToken, randomVal, trustScore, ip, fairMode)
}

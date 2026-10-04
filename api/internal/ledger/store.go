// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/ledger/store.go
// PURPOSE: Persistence layer for the hash-chained ledger
// INPUTS / OUTPUTS: Appends events to ledger:events, verifies the chain
// DEPENDS ON: event.go
// USED BY: service.go
// RULES: Chain append must be atomic; canonical encoding must be byte-stable
// DO NOT: N/A
package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fairdrop/api/internal/store"
)

// unitSep is the field delimiter for the canonical string. Using a control
// character means no field value can forge a boundary between two fields.
const unitSep = "\x1f"

type Store struct {
	rdb *store.RedisClient
}

func NewStore(rdb *store.RedisClient) *Store {
	return &Store{rdb: rdb}
}

// CanonicalString renders the event as a delimiter-joined field sequence with a
// FIXED field order.
//
// This must be byte-identical to the Lua assembler below or verification will
// fail. JSON encoding is deliberately avoided here: cjson in Redis and
// encoding/json in Go do not agree on map key ordering or float formatting, so
// a JSON-based canonical form can never be verified across the two runtimes.
func (e *Event) CanonicalString() string {
	meta := "{}"
	if len(e.Metadata) > 0 {
		if b, err := json.Marshal(e.Metadata); err == nil {
			meta = string(b)
		}
	}
	fields := []string{
		strconv.FormatInt(e.ID, 10),
		e.PrevHash,
		e.Timestamp.UTC().Format(time.RFC3339Nano),
		e.UserID,
		e.UserType,
		e.HumanProfile,
		e.IP,
		e.Subnet,
		strconv.FormatFloat(e.TrustScore, 'f', 6, 64),
		e.EventType,
		e.ReasonCode,
		meta,
	}
	return strings.Join(fields, unitSep)
}

// RecordEvent appends an event to the hash chain, atomically.
//
// The head read, hash computation and LPUSH must all happen inside ONE Lua
// script. This is a correctness requirement, not an optimisation: if the head
// were read in Go first, two concurrent writers would both observe the same
// PrevHash and emit two events claiming the same parent, forking the chain and
// breaking verification from that point onward.
//
// ARGV: [1]=id [2]=prev_hash_unused [3]=timestamp [4]=user_id [5]=user_type
//
//	[6]=human_profile [7]=ip [8]=subnet [9]=trust [10]=event_type
//	[11]=reason_code [12]=metadata_json [13]=genesis_hash [14]=seq_key
func (s *Store) RecordEvent(ctx context.Context, event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	metaJSON := "{}"
	if len(event.Metadata) > 0 {
		if b, err := json.Marshal(event.Metadata); err == nil {
			metaJSON = string(b)
		}
	}

	const sep = "\x1f"
	script := `
			local prev = redis.call('get', KEYS[2])
			if not prev then prev = ARGV[13] end

			-- Monotonic sequence number assigned at write time, so event IDs are
			-- unique and ordered without a second round-trip.
			local seq = redis.call('incr', KEYS[3])

			-- Field order here MUST match Event.CanonicalString() in Go.
			local canonical = table.concat({
				tostring(seq),
				prev,
				ARGV[3],
				ARGV[4],
				ARGV[5],
				ARGV[6],
				ARGV[7],
				ARGV[8],
				ARGV[9],
				ARGV[10],
				ARGV[11],
				ARGV[12]
			}, ARGV[14])

			local hash = redis.sha1hex(prev .. canonical)

			local payload = '{"id":' .. tostring(seq) ..
				',"prev_hash":"' .. prev ..
				'","hash":"' .. hash ..
				'","timestamp":"' .. ARGV[3] ..
				'","user_id":"' .. ARGV[4] ..
				'","user_type":"' .. ARGV[5] ..
				'","human_profile":"' .. ARGV[6] ..
				'","ip":"' .. ARGV[7] ..
				'","subnet":"' .. ARGV[8] ..
				'","trust_score":' .. ARGV[9] ..
				',"event_type":"' .. ARGV[10] ..
				'","reason_code":"' .. ARGV[11] ..
				'","metadata":' .. ARGV[12] .. '}'

			redis.call('lpush', KEYS[1], payload)
			redis.call('set', KEYS[2], hash)
			return hash
		`

	return s.rdb.Client.Eval(ctx, script,
		[]string{"ledger:events", ChainStateKey, ChainSeqKey},
		event.ID,
		"",
		event.Timestamp.UTC().Format(time.RFC3339Nano),
		event.UserID,
		event.UserType,
		event.HumanProfile,
		event.IP,
		event.Subnet,
		strconv.FormatFloat(event.TrustScore, 'f', 6, 64),
		event.EventType,
		event.ReasonCode,
		metaJSON,
		GenesisHash,
		sep,
	).Err()
}

// ChainVerification is the result of walking the hash chain.
type ChainVerification struct {
	OK       bool   `json:"ok"`
	Total    int64  `json:"total_events"`
	Checked  int    `json:"checked"`
	HeadHash string `json:"head_hash,omitempty"`
	BrokenAt int64  `json:"broken_at_index,omitempty"`
	Error    string `json:"error,omitempty"`
}

// VerifyChain walks stored events oldest-first and reports the first break.
// Events are LPUSH'd, so index 0 is the NEWEST; chronological order is the
// reverse.
func (s *Store) VerifyChain(ctx context.Context, limit int64) (ChainVerification, error) {
	v := ChainVerification{OK: true}

	length, err := s.rdb.Client.LLen(ctx, "ledger:events").Result()
	if err != nil {
		return v, err
	}
	v.Total = length
	if length == 0 {
		return v, nil
	}

	if limit <= 0 || limit > length {
		limit = length
	}

	raw, err := s.rdb.Client.LRange(ctx, "ledger:events", length-limit, length-1).Result()
	if err != nil {
		return v, err
	}

	// Reverse into chronological order (index 0 of the slice is the newest).
	ordered := make([]Event, 0, len(raw))
	for i := len(raw) - 1; i >= 0; i-- {
		var e Event
		if err := json.Unmarshal([]byte(raw[i]), &e); err != nil {
			v.OK = false
			v.BrokenAt = int64(i + 1)
			v.Error = "unparseable event: " + err.Error()
			return v, nil
		}
		ordered = append(ordered, e)
	}
	v.Checked = len(ordered)

	// A partial view cannot verify the anchor link, only the links it can see.
	partial := length > limit

	for i, e := range ordered {
		idx := int64(i + 1)

		// Anchor check: the first event of a complete chain must cite genesis.
		if i == 0 {
			if !partial && e.PrevHash != GenesisHash {
				v.OK = false
				v.BrokenAt = idx
				v.Error = "chain does not start at genesis"
				return v, nil
			}
		} else if e.PrevHash != ordered[i-1].Hash {
			v.OK = false
			v.BrokenAt = idx
			v.Error = "broken link: prev_hash does not match preceding event hash"
			return v, nil
		}

		if recomputed := e.ComputeHash(); recomputed != e.Hash {
			v.OK = false
			v.BrokenAt = idx
			v.Error = "content hash mismatch: event modified after write"
			return v, nil
		}
	}

	if len(ordered) > 0 {
		v.HeadHash = ordered[len(ordered)-1].Hash
	}
	return v, nil
}

// GetRecentEvents returns the newest `limit` events, newest first.
func (s *Store) GetRecentEvents(ctx context.Context, limit int64) ([]Event, error) {
	key := "ledger:events"
	results, err := s.rdb.Client.LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(results))
	for _, res := range results {
		var event Event
		if err := json.Unmarshal([]byte(res), &event); err == nil {
			events = append(events, event)
		}
	}
	return events, nil
}

// GetAllEvents returns every stored event in chronological order.
// LPUSH prepends, so index 0 is the NEWEST; reading the whole list and
// reversing in Go is unambiguous, unlike trying to ask Redis for a reversed
// range.
func (s *Store) GetAllEvents(ctx context.Context) ([]Event, error) {
	length, err := s.rdb.Client.LLen(ctx, "ledger:events").Result()
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []Event{}, nil
	}
	raw, err := s.rdb.Client.LRange(ctx, "ledger:events", 0, length-1).Result()
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(raw))
	for i := len(raw) - 1; i >= 0; i-- {
		var e Event
		if err := json.Unmarshal([]byte(raw[i]), &e); err == nil {
			events = append(events, e)
		}
	}
	return events, nil
}

var _ = fmt.Sprintf

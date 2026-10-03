// FILE: d:/hacks/BitNBuilds/sim/internal/report/report.go
// PURPOSE: Report generator for simulation metrics, confusion matrix, and ledger cross-checks
// INPUTS / OUTPUTS: Full simulation JSON report matching SIMULATOR_SPEC.md §7
// DEPENDS ON: N/A
// USED BY: sim/runner, sim/control
// RULES: Compute TP, FP, FN, TN, precision, recall, and fairness score
// DO NOT: Alter metrics reporting format

package report

import (
	"sort"
	"sync"
	"time"
)

type ResultMetric struct {
	UserType string        `json:"user_type"`
	Profile  string        `json:"profile"`
	Status   int           `json:"status"`
	Latency  time.Duration `json:"latency"`
	Claimed  bool          `json:"claimed"`
	Reason   string        `json:"reason"`
}

type ProfileReport struct {
	Requests int     `json:"requests"`
	Accepted int     `json:"accepted"`
	Rejected int     `json:"rejected"`
	Hits429  int     `json:"hits_429"`
	WinRate  float64 `json:"win_rate"`
}

type SimulationReport struct {
	Scenario                string                   `json:"scenario"`
	Mode                    string                   `json:"mode"`
	Seed                    int64                    `json:"seed"`
	Seats                   int                      `json:"seats"`
	TotalRequests           int                      `json:"total_requests"`
	HumanRequests           int                      `json:"human_requests"`
	BotRequests             int                      `json:"bot_requests"`
	HumanClaims             int                      `json:"human_claims"`
	BotClaims               int                      `json:"bot_claims"`
	HumanSuccessRate        float64                  `json:"human_success_rate"`
	BotSuccessRate          float64                  `json:"bot_success_rate"`
	FairnessScore           float64                  `json:"fairness_score"`
	RateLimiterHits         int                      `json:"rate_limiter_hits"`
	LatencyP50Ms            float64                  `json:"latency_p50_ms"`
	LatencyP95Ms            float64                  `json:"latency_p95_ms"`
	LatencyP99Ms            float64                  `json:"latency_p99_ms"`
	TP                      int                      `json:"tp"` // Bots blocked
	FN                      int                      `json:"fn"` // Bots let through
	FP                      int                      `json:"fp"` // Humans wrongly rejected
	TN                      int                      `json:"tn"` // Humans accepted
	Precision               float64                  `json:"precision"`
	Recall                  float64                  `json:"recall"`
	HumanFalseRejectionRate float64                  `json:"human_false_rejection_rate"`
	OversellCount           int                      `json:"oversell_count"`
	DuplicateCount          int                      `json:"duplicate_count"`
	Profiles                map[string]ProfileReport `json:"profiles"`
	LedgerCrossCheck        map[string]interface{}   `json:"ledger_cross_check,omitempty"`
}

type Collector struct {
	mu      sync.Mutex
	metrics []ResultMetric
}

func NewCollector() *Collector {
	return &Collector{
		metrics: make([]ResultMetric, 0, 2000),
	}
}

func (c *Collector) Record(metric ResultMetric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, metric)
}

func (c *Collector) GenerateReport(scenario, mode string, seed int64, seats int) SimulationReport {
	c.mu.Lock()
	defer c.mu.Unlock()

	var r SimulationReport
	r.Scenario = scenario
	r.Mode = mode
	r.Seed = seed
	r.Seats = seats
	r.TotalRequests = len(c.metrics)
	r.Profiles = make(map[string]ProfileReport)

	if r.TotalRequests == 0 {
		return r
	}

	var latencies []float64
	var humanCount, botCount int
	var humanSuccess, botSuccess int

	for _, m := range c.metrics {
		latMs := float64(m.Latency.Milliseconds())
		latencies = append(latencies, latMs)

		profRep := r.Profiles[m.Profile]
		profRep.Requests++

		if m.Status == 429 {
			r.RateLimiterHits++
			profRep.Hits429++
		}

		if m.UserType == "human" {
			humanCount++
			if m.Claimed {
				humanSuccess++
				r.TN++ // Human accepted
				profRep.Accepted++
			} else {
				r.FP++ // Human rejected/blocked
				profRep.Rejected++
			}
		} else {
			botCount++
			if m.Claimed {
				botSuccess++
				r.FN++ // Bot let through
				profRep.Accepted++
			} else {
				r.TP++ // Bot blocked
				profRep.Rejected++
			}
		}

		r.Profiles[m.Profile] = profRep
	}

	for k, v := range r.Profiles {
		if v.Requests > 0 {
			v.WinRate = (float64(v.Accepted) / float64(v.Requests)) * 100.0
			r.Profiles[k] = v
		}
	}

	r.HumanRequests = humanCount
	r.BotRequests = botCount
	r.HumanClaims = humanSuccess
	r.BotClaims = botSuccess

	if humanCount > 0 {
		r.HumanSuccessRate = (float64(humanSuccess) / float64(humanCount)) * 100.0
		r.HumanFalseRejectionRate = (float64(r.FP) / float64(humanCount)) * 100.0
	}
	if botCount > 0 {
		r.BotSuccessRate = (float64(botSuccess) / float64(botCount)) * 100.0
	}

	if r.BotSuccessRate > 0 {
		r.FairnessScore = (r.HumanSuccessRate / r.BotSuccessRate) * 100.0
	} else if r.HumanSuccessRate > 0 {
		r.FairnessScore = 100.0
	}

	if (r.TP + r.FP) > 0 {
		r.Precision = float64(r.TP) / float64(r.TP+r.FP)
	}
	if (r.TP + r.FN) > 0 {
		r.Recall = float64(r.TP) / float64(r.TP+r.FN)
	}

	if len(latencies) > 0 {
		sort.Float64s(latencies)
		r.LatencyP50Ms = quantile(latencies, 0.50)
		r.LatencyP95Ms = quantile(latencies, 0.95)
		r.LatencyP99Ms = quantile(latencies, 0.99)
	}

	return r
}

func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

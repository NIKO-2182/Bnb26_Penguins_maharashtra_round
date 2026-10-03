// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/sim/internal/profiles/profiles.go
// PURPOSE: Profile definitions for simulator clients
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: sim/runner
// RULES: N/A
// DO NOT: N/A
package profiles

type ProfileConfig struct {
	Name         string `json:"name"`
	UserType     string `json:"user_type"`     // "human" | "bot"
	HumanProfile string `json:"human_profile"` // "human_normal" | "human_slow" | "human_frustrated" | "human_shared_ip"
	PoWSolveMs   int    `json:"pow_solve_ms"`
	ClickDelayMs int    `json:"click_delay_ms"`
	BurstCount   int    `json:"burst_count"`
	RetryCount   int    `json:"retry_count"`
}

var DefaultProfiles = map[string]ProfileConfig{
	"human_normal": {
		Name:         "human_normal",
		UserType:     "human",
		HumanProfile: "human_normal",
		PoWSolveMs:   500,
		ClickDelayMs: 2000,
		BurstCount:   1,
		RetryCount:   2,
	},
	"human_slow": {
		Name:         "human_slow",
		UserType:     "human",
		HumanProfile: "human_slow",
		PoWSolveMs:   1200,
		ClickDelayMs: 4000,
		BurstCount:   1,
		RetryCount:   1,
	},
	"human_frustrated": {
		Name:         "human_frustrated",
		UserType:     "human",
		HumanProfile: "human_frustrated",
		PoWSolveMs:   400,
		ClickDelayMs: 150,
		BurstCount:   8,
		RetryCount:   5,
	},
	"human_shared_ip": {
		Name:         "human_shared_ip",
		UserType:     "human",
		HumanProfile: "human_shared_ip",
		PoWSolveMs:   600,
		ClickDelayMs: 1800,
		BurstCount:   2,
		RetryCount:   3,
	},
	"bot_naive": {
		Name:         "bot_naive",
		UserType:     "bot",
		HumanProfile: "",
		PoWSolveMs:   10,
		ClickDelayMs: 20,
		BurstCount:   50,
		RetryCount:   10,
	},
	"bot_solver": {
		Name:         "bot_solver",
		UserType:     "bot",
		HumanProfile: "",
		PoWSolveMs:   5,
		ClickDelayMs: 50,
		BurstCount:   20,
		RetryCount:   5,
	},
	"bot_distributed": {
		Name:         "bot_distributed",
		UserType:     "bot",
		HumanProfile: "",
		PoWSolveMs:   100,
		ClickDelayMs: 500,
		BurstCount:   5,
		RetryCount:   5,
	},
	"bot_retry": {
		Name:         "bot_retry",
		UserType:     "bot",
		HumanProfile: "",
		PoWSolveMs:   10,
		ClickDelayMs: 10,
		BurstCount:   100,
		RetryCount:   20,
	},
}

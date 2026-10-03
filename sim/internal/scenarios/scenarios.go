// FILE: d:/hacks/BitNBuilds/sim/internal/scenarios/scenarios.go
// PURPOSE: Preset scenario configs matching SIMULATOR_SPEC.md §6
// INPUTS / OUTPUTS: Returns scenario configuration presets
// DEPENDS ON: N/A
// USED BY: sim/internal/control
// RULES: Support baseline_no_bots, flash_sale, late_arrivals, shared_ip, sweep, unseen_attack
// DO NOT: N/A

package scenarios

type ScenarioConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Seats       int    `json:"seats"`
	BotsCount   int    `json:"bots_count"`
	HumansCount int    `json:"humans_count"`
	Mode        string `json:"mode"`
}

var DefaultScenarios = map[string]ScenarioConfig{
	"baseline_no_bots": {
		Name:        "baseline_no_bots",
		Description: "Baseline: Humans only to confirm normal operation and human win rate",
		Seats:       100,
		BotsCount:   0,
		HumansCount: 100,
		Mode:        "fair",
	},
	"flash_sale": {
		Name:        "flash_sale",
		Description: "Flash Sale: 500 seats, 2500 humans, 25000 bots simultaneous attack",
		Seats:       500,
		BotsCount:   2500,
		HumansCount: 500,
		Mode:        "fair",
	},
	"late_arrivals": {
		Name:        "late_arrivals",
		Description: "Late Arrivals: Portion of humans join near end of window",
		Seats:       200,
		BotsCount:   500,
		HumansCount: 200,
		Mode:        "fair",
	},
	"shared_ip": {
		Name:        "shared_ip",
		Description: "Shared IP: 200 humans behind 5 IPs under bot attack",
		Seats:       100,
		BotsCount:   1000,
		HumansCount: 200,
		Mode:        "fair",
	},
	"unseen_attack": {
		Name:        "unseen_attack",
		Description: "Unseen Attack: Distributed bot attack profile evaluation",
		Seats:       200,
		BotsCount:   1000,
		HumansCount: 100,
		Mode:        "fair",
	},
}

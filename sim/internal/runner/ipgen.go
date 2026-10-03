// FILE: d:/hacks/BitNBuilds/sim/internal/runner/ipgen.go
// PURPOSE: IP pool generator for single, shared, and distributed user scenarios
// INPUTS / OUTPUTS: Returns deterministic IP addresses per user profile
// DEPENDS ON: N/A
// USED BY: sim/internal/runner/runner.go
// RULES: Generate subnet-partitioned IP addresses
// DO NOT: N/A

package runner

import (
	"fmt"
)

type IPGenerator struct {
	sharedHumanIPs []string
}

func NewIPGenerator() *IPGenerator {
	shared := make([]string, 5)
	for i := 0; i < 5; i++ {
		shared[i] = fmt.Sprintf("198.51.100.%d", 10+i)
	}
	return &IPGenerator{
		sharedHumanIPs: shared,
	}
}

func (g *IPGenerator) GetIP(userType, profileName string, index int) string {
	switch profileName {
	case "human_shared_ip":
		return g.sharedHumanIPs[index%len(g.sharedHumanIPs)]
	case "bot_naive":
		return "203.0.113.1" // Single IP flood
	case "bot_distributed":
		return fmt.Sprintf("172.16.%d.%d", (index/250)%255, (index%250)+1)
	case "bot_solver", "bot_retry":
		return fmt.Sprintf("192.0.2.%d", (index%200)+1)
	default:
		if userType == "human" {
			return fmt.Sprintf("192.168.%d.%d", (index/200)+1, (index%200)+1)
		}
		return fmt.Sprintf("10.100.%d.%d", (index/200)+1, (index%200)+1)
	}
}

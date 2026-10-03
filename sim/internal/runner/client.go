// FILE: d:/hacks/BitNBuilds/sim/internal/runner/client.go
// PURPOSE: Full-flow HTTP client (join, solve PoW, verify, poll session, claim)
// INPUTS / OUTPUTS: Executes sequential API protocol calls and logs outcomes
// DEPENDS ON: sim/internal/profiles, sim/internal/report, sim/internal/runner/pow.go
// USED BY: sim/internal/runner/runner.go
// RULES: Must execute full protocol flow without bypassing join/verify steps
// DO NOT: Call /claim directly without session handshake

package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"fairdrop/sim/internal/profiles"
	"fairdrop/sim/internal/report"
)

type FullFlowClient struct {
	client    *http.Client
	baseURL   string
	powSolver *PoWSolver
}

func NewFullFlowClient(baseURL string, client *http.Client) *FullFlowClient {
	return &FullFlowClient{
		client:    client,
		baseURL:   baseURL,
		powSolver: NewPoWSolver(),
	}
}

func (c *FullFlowClient) Execute(ctx context.Context, prof profiles.ProfileConfig, userID, clientIP string, collector *report.Collector) {
	startTotal := time.Now()

	// Step 1: POST /join
	joinPayload, _ := json.Marshal(map[string]string{"username": userID})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/join", bytes.NewBuffer(joinPayload))
	if err != nil {
		collector.Record(report.ResultMetric{UserType: prof.UserType, Profile: prof.Name, Status: 500, Latency: time.Since(startTotal), Claimed: false})
		return
	}
	c.setHeaders(req, prof, clientIP, "")

	resp, err := c.client.Do(req)
	if err != nil || resp == nil {
		collector.Record(report.ResultMetric{UserType: prof.UserType, Profile: prof.Name, Status: 500, Latency: time.Since(startTotal), Claimed: false})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		collector.Record(report.ResultMetric{UserType: prof.UserType, Profile: prof.Name, Status: 429, Latency: time.Since(startTotal), Claimed: false})
		return
	}

	var joinRes struct {
		SessionToken string `json:"session_token"`
		Token        string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&joinRes)

	token := joinRes.SessionToken
	if token == "" {
		token = joinRes.Token
	}
	if token == "" {
		token = userID
	}

	// Step 2: Skip PoW if bot_naive
	if prof.Name != "bot_naive" {
		solveDuration := time.Duration(prof.PoWSolveMs) * time.Millisecond
		solution := c.powSolver.Solve("challenge", 4, solveDuration)

		// Step 3: POST /verify
		vBody, _ := json.Marshal(map[string]string{
			"session_token": token,
			"solution":      solution,
		})
		vReq, vErr := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/verify", bytes.NewBuffer(vBody))
		if vErr == nil {
			c.setHeaders(vReq, prof, clientIP, token)
			vResp, vDoErr := c.client.Do(vReq)
			if vDoErr == nil && vResp != nil {
				vResp.Body.Close()
			}
		}
	}

	// Step 4: GET /session (polling verification)
	sReq, sErr := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/session?token=%s", c.baseURL, token), nil)
	if sErr == nil {
		c.setHeaders(sReq, prof, clientIP, token)
		sResp, sDoErr := c.client.Do(sReq)
		if sDoErr == nil && sResp != nil {
			sResp.Body.Close()
		}
	}

	// Step 5: Click delay before POST /claim
	if prof.ClickDelayMs > 0 {
		time.Sleep(time.Duration(prof.ClickDelayMs) * time.Millisecond)
	}

	// Step 6: POST /claim (with optional retries for frustrated humans or retry bots)
	attempts := prof.RetryCount
	if attempts < 1 {
		attempts = 1
	}

	for a := 0; a < attempts; a++ {
		if a > 0 && prof.Name == "human_frustrated" {
			time.Sleep(50 * time.Millisecond) // Short irregular retry burst
		}

		cBody, _ := json.Marshal(map[string]string{"claim_token": token})
		cReq, cErr := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/claim", bytes.NewBuffer(cBody))
		if cErr != nil {
			break
		}
		c.setHeaders(cReq, prof, clientIP, token)

		cStart := time.Now()
		cResp, cDoErr := c.client.Do(cReq)
		cLat := time.Since(cStart)

		if cDoErr != nil || cResp == nil {
			collector.Record(report.ResultMetric{
				UserType: prof.UserType,
				Profile:  prof.Name,
				Status:   500,
				Latency:  cLat,
				Claimed:  false,
			})
			continue
		}

		var claimRes struct {
			Status string `json:"status"`
		}
		_ = json.NewDecoder(cResp.Body).Decode(&claimRes)
		cResp.Body.Close()

		isClaimed := cResp.StatusCode == http.StatusOK && (claimRes.Status == "seat_granted" || claimRes.Status == "claimed" || claimRes.Status == "allocated")

		collector.Record(report.ResultMetric{
			UserType: prof.UserType,
			Profile:  prof.Name,
			Status:   cResp.StatusCode,
			Latency:  cLat,
			Claimed:  isClaimed,
			Reason:   claimRes.Status,
		})

		if isClaimed || cResp.StatusCode == 409 { // Sold out or claimed
			break
		}
	}
}

func (c *FullFlowClient) setHeaders(req *http.Request, prof profiles.ProfileConfig, clientIP, token string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("X-Sim-IP", clientIP)
	req.Header.Set("X-Sim-User-Type", prof.UserType)
	req.Header.Set("X-Sim-Profile", prof.Name)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

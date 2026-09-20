package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/mueedx/job-bot/backend/internal/models"
)

var blockchainKeywords = []string{
	"solidity", "evm", "smart contract", "smart contracts", "web3", "defi",
	"chainlink", "foundry", "hardhat", "substrate", "bittensor", "dtao",
	"dapp", "indexer", "subgraphs", "subgraph",
}

var fdeKeywords = []string{
	"forward deployed", "fde", "ai engineer", "agentic", "agents", "mcp",
	"model context protocol", "a2a", "llm", "prompt engineering",
	"solutions architect", "customer engineer", "rag",
}

var fullstackKeywords = []string{
	"next.js", "nextjs", "react", "nestjs", "node.js", "nodejs", "typescript",
	"prisma", "mysql", "postgresql", "postgres", "rest api", "full stack",
	"fullstack", "frontend", "backend",
}

var customerFacing = []string{
	"customer", "solutions", "forward deployed", "fde", "client-facing",
	"architecture", "deployed engineer",
}

var protocolHeavy = []string{
	"smart contract", "solidity", "protocol", "evm", "on-chain", "mainnet",
	"indexer", "subgraph",
}

// MatchResult is the output of scoring a job.
type MatchResult struct {
	Score         float64
	Track         string
	MatchedSkills []string
	MissingSkills []string
	ScoreReasons  string
	StatusAfter   string // scored | queued
}

// MatchJob scores a job and picks a resume track.
func MatchJob(job *models.Job) MatchResult {
	blob := strings.ToLower(job.Title + "\n" + job.Description + "\n" + nullStr(job.Location))
	bcHits := hitKeywords(blob, blockchainKeywords)
	fdeHits := hitKeywords(blob, fdeKeywords)
	fsHits := hitKeywords(blob, fullstackKeywords)

	track := "fullstack"
	matched := fsHits
	switch {
	case len(bcHits) > 0 && len(fdeHits) > 0:
		if hasAny(blob, customerFacing) && !hasAny(blob, protocolHeavy) {
			track = "fde"
			matched = unique(append(fdeHits, bcHits...))
		} else if hasAny(blob, protocolHeavy) {
			track = "blockchain"
			matched = unique(append(bcHits, fdeHits...))
		} else if hasAny(blob, customerFacing) {
			track = "fde"
			matched = unique(append(fdeHits, bcHits...))
		} else {
			track = "blockchain"
			matched = unique(append(bcHits, fdeHits...))
		}
	case len(bcHits) >= len(fdeHits) && len(bcHits) >= len(fsHits) && len(bcHits) > 0:
		track = "blockchain"
		matched = bcHits
	case len(fdeHits) >= len(fsHits) && len(fdeHits) > 0:
		track = "fde"
		matched = fdeHits
	case len(fsHits) > 0:
		track = "fullstack"
		matched = fsHits
	}

	score := 0.35
	score += math.Min(0.35, float64(len(matched))*0.07)

	if job.IsRemote || looksRemoteLoc(nullStr(job.Location)) {
		score += 0.12
	} else if job.IsRelocation {
		score += 0.05
	} else {
		score -= 0.08
	}

	score += compensationScore(job.SalaryMin, job.SalaryMax)

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	missing := suggestMissing(track, matched)
	reason := fmt.Sprintf(
		"Routed to %s from keyword signal (%d hits). Remote/comp heuristics adjusted fit to %.0f%%.",
		track, len(matched), score*100,
	)

	threshold := autoApplyMinScore()
	status := "scored"
	if score >= threshold {
		status = "queued"
	}

	return MatchResult{
		Score:         score,
		Track:         track,
		MatchedSkills: matched,
		MissingSkills: missing,
		ScoreReasons:  reason,
		StatusAfter:   status,
	}
}

func SkillsJSON(skills []string) string {
	b, _ := json.Marshal(skills)
	return string(b)
}

func autoApplyMinScore() float64 {
	raw := os.Getenv("AUTO_APPLY_MIN_SCORE")
	if raw == "" {
		return 0.80
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0.80
	}
	return v
}

func draftThreshold() float64 {
	raw := os.Getenv("DRAFT_MIN_SCORE")
	if raw == "" {
		return 0.75
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0.75
	}
	return v
}

// DraftMinScore is exported for ingest/telegram.
func DraftMinScore() float64 { return draftThreshold() }

// NotifyMinScore for telegram alerts (same default as draft).
func NotifyMinScore() float64 { return draftThreshold() }

func compensationScore(min, max *int) float64 {
	if min == nil && max == nil {
		return 0.05 // neutral-ish
	}
	lo := 0
	hi := 0
	if min != nil {
		lo = *min
	}
	if max != nil {
		hi = *max
	}
	if hi == 0 {
		hi = lo
	}
	if lo == 0 {
		lo = hi
	}
	mid := (lo + hi) / 2
	if mid > 0 && mid < 1800 {
		return -0.25
	}
	if mid >= 2500 && mid <= 4000 {
		return 0.15
	}
	if mid > 4000 {
		return 0.10
	}
	return 0.0
}

func hitKeywords(blob string, keys []string) []string {
	var hits []string
	for _, k := range keys {
		if strings.Contains(blob, strings.ToLower(k)) {
			hits = append(hits, k)
		}
	}
	return hits
}

func hasAny(blob string, keys []string) bool {
	return len(hitKeywords(blob, keys)) > 0
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func suggestMissing(track string, matched []string) []string {
	pool := fullstackKeywords
	switch track {
	case "blockchain":
		pool = blockchainKeywords
	case "fde":
		pool = fdeKeywords
	}
	have := map[string]bool{}
	for _, m := range matched {
		have[strings.ToLower(m)] = true
	}
	var miss []string
	for _, k := range pool {
		if !have[strings.ToLower(k)] {
			miss = append(miss, k)
		}
		if len(miss) >= 4 {
			break
		}
	}
	return miss
}

func nullStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func looksRemoteLoc(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "remote") || strings.Contains(l, "anywhere")
}

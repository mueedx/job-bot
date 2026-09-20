package services_test

import (
	"testing"

	"github.com/mueedx/job-bot/backend/internal/models"
	"github.com/mueedx/job-bot/backend/internal/services"
)

func TestMatchJob_Fullstack(t *testing.T) {
	job := &models.Job{
		Title:       "Senior Full Stack Engineer",
		Description: "Build with Next.js, NestJS, TypeScript, Prisma and PostgreSQL.",
		IsRemote:    true,
	}
	r := services.MatchJob(job)
	if r.Track != "fullstack" {
		t.Fatalf("track=%s want fullstack", r.Track)
	}
	if r.Score < 0.4 {
		t.Fatalf("score too low: %v", r.Score)
	}
}

func TestMatchJob_Blockchain(t *testing.T) {
	job := &models.Job{
		Title:       "Smart Contract Engineer",
		Description: "Solidity, EVM, Foundry, Hardhat, DeFi protocols and subgraphs.",
		IsRemote:    true,
	}
	r := services.MatchJob(job)
	if r.Track != "blockchain" {
		t.Fatalf("track=%s want blockchain", r.Track)
	}
}

func TestMatchJob_FDETieBreak(t *testing.T) {
	job := &models.Job{
		Title:       "Forward Deployed Engineer — AI/Web3",
		Description: "Customer-facing solutions architect role using MCP, agents, LLM and Bittensor subnet tooling.",
		IsRemote:    true,
	}
	r := services.MatchJob(job)
	if r.Track != "fde" {
		t.Fatalf("track=%s want fde (customer-facing AI+Web3)", r.Track)
	}
}

func TestMatchJob_ProtocolHeavyTieBreak(t *testing.T) {
	job := &models.Job{
		Title:       "Protocol Engineer",
		Description: "Bittensor agents and AI with heavy Solidity smart contracts, EVM, indexer and mainnet deployments.",
		IsRemote:    true,
	}
	r := services.MatchJob(job)
	if r.Track != "blockchain" {
		t.Fatalf("track=%s want blockchain (protocol-heavy)", r.Track)
	}
}

package services

import (
	"testing"

	"github.com/mueedx/job-bot/backend/internal/models"
)

func eligibleJob(title, desc string) *models.Job {
	return &models.Job{Title: title, Description: desc, IsRemote: true}
}

func TestEvaluateEligibilityRules(t *testing.T) {
	cases := []struct {
		name    string
		job     *models.Job
		rules   func(*EligibilityRules)
		status  string
		rule    string
		wantRel bool
	}{
		{
			name:   "worldwide remote passes",
			job:    eligibleJob("Backend Engineer", "We are a distributed team, work from anywhere."),
			status: EligibilityPass,
			rule:   RuleWorldwideRemote,
		},
		{
			name:   "hire anywhere via EOR passes",
			job:    eligibleJob("Platform Engineer", "We hire anywhere via our employer of record."),
			status: EligibilityPass,
			rule:   RuleWorldwideRemote,
		},
		{
			name:    "sponsorship plus relocation passes and highlights",
			job:     eligibleJob("Senior Engineer", "We offer visa sponsorship and relocation assistance."),
			status:  EligibilityPass,
			rule:    RuleSponsoredRelocation,
			wantRel: true,
		},
		{
			name:   "sponsorship without relocation does not highlight",
			job:    eligibleJob("Senior Engineer", "We offer visa sponsorship."),
			status: EligibilityUnknown,
			rule:   RuleNoSignal,
		},
		{
			name:   "country-bounded remote is vetoed",
			job:    eligibleJob("Developer", "Remote US only. You must reside in the United States."),
			status: EligibilityVeto,
			rule:   RuleCountryBoundedRemote,
		},
		{
			name: "country-bounded remote passes with work authorization",
			job:  eligibleJob("Developer", "Remote US only."),
			rules: func(r *EligibilityRules) {
				r.WorkAuthorizedCountries = []string{"us"}
			},
			status: EligibilityPass,
			rule:   RuleAuthorizedCountry,
		},
		{
			name: "work-authorized exemption can be disabled",
			job:  eligibleJob("Developer", "Remote US only."),
			rules: func(r *EligibilityRules) {
				r.WorkAuthorizedCountries = []string{"us"}
				r.HonorWorkAuthorizedCountries = boolPtr(false)
			},
			status: EligibilityVeto,
			rule:   RuleCountryBoundedRemote,
		},
		{
			name:   "local work permit requirement is vetoed",
			job:    eligibleJob("Developer", "Must have the right to work in Germany. No visa sponsorship."),
			status: EligibilityVeto,
			rule:   RuleLocalWorkPermit,
		},
		{
			name:   "citizenship requirement is vetoed",
			job:    eligibleJob("Developer", "Applicants must be a US citizen."),
			status: EligibilityVeto,
			rule:   RuleLocalWorkPermit,
		},
		{
			name:   "no signal is unknown and passes by default",
			job:    eligibleJob("Engineer", "Build great software with us."),
			status: EligibilityUnknown,
			rule:   RuleNoSignal,
		},
		{
			name:   "no signal is vetoed when configured",
			job:    eligibleJob("Engineer", "Build great software with us."),
			rules:  func(r *EligibilityRules) { r.VetoUnknown = boolPtr(true) },
			status: EligibilityVeto,
			rule:   RuleUnknownVetoByOperator,
		},
		{
			name:   "everything passes when the engine is off",
			job:    eligibleJob("Developer", "Remote US only. Citizens only."),
			rules:  func(r *EligibilityRules) { r.Enabled = boolPtr(false) },
			status: EligibilityPass,
			rule:   RuleDisabled,
		},
		{
			name:   "rule can be individually disabled",
			job:    eligibleJob("Developer", "Remote UK only."),
			rules:  func(r *EligibilityRules) { r.VetoCountryBoundedRemote = boolPtr(false) },
			status: EligibilityUnknown,
			rule:   RuleNoSignal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules := DefaultEligibilityRules()
			if tc.rules != nil {
				tc.rules(rules)
			}
			v := EvaluateEligibility(tc.job, rules)
			if v.Status != tc.status {
				t.Fatalf("status = %q, want %q (reason: %s)", v.Status, tc.status, v.Reason)
			}
			if v.Rule != tc.rule {
				t.Fatalf("rule = %q, want %q", v.Rule, tc.rule)
			}
			if v.HighlightsRelocation != tc.wantRel {
				t.Fatalf("highlights_relocation = %v, want %v", v.HighlightsRelocation, tc.wantRel)
			}
		})
	}
}

func TestEligibilityOnSiteSponsorshipRequiresBoth(t *testing.T) {
	// On-site roles only pass via the sponsorship + relocation route, so an
	// on-site posting with neither signal must not pass.
	job := &models.Job{Title: "Java Dev", Description: "Visa sponsorship available.", IsRemote: false}
	v := EvaluateEligibility(job, DefaultEligibilityRules())
	if v.Status != EligibilityUnknown {
		t.Fatalf("on-site with sponsorship only: status = %q, want unknown", v.Status)
	}
}

func TestEligibilityRulesMergeKeepsDefaults(t *testing.T) {
	rules := DefaultEligibilityRules()
	rules.Merge(&EligibilityRules{WorldwideRemoteOK: boolPtr(false)})
	rules.Normalize()

	if on(rules.WorldwideRemoteOK) {
		t.Fatal("merged override lost")
	}
	if !on(rules.SponsoredRelocationOK) {
		t.Fatal("unmentioned flag lost its default")
	}
	if len(rules.WorldwidePhrases) == 0 {
		t.Fatal("unmentioned phrase list lost its defaults")
	}
}

func TestEligibilityVetoNeverDrafts(t *testing.T) {
	// A veto must map to the archived/rejected status regardless of score,
	// so the drafter never sees the job.
	job := eligibleJob("Developer", "Remote Poland only.")
	v := EvaluateEligibility(job, DefaultEligibilityRules())
	if !v.Vetoed() {
		t.Fatalf("expected veto, got %q", v.Status)
	}
	if got := v.StatusAfter(0.99); got != StatusRejected {
		t.Fatalf("StatusAfter = %q, want %q", got, StatusRejected)
	}
}

func TestEligibilityPhraseListsEditable(t *testing.T) {
	rules := DefaultEligibilityRules()
	rules.WorldwidePhrases = []string{"moonlight from mars"}
	rules.Normalize()

	v := EvaluateEligibility(eligibleJob("Eng", "Everyone is welcome to moonlight from Mars with us."), rules)
	if v.Status != EligibilityPass || v.Rule != RuleWorldwideRemote {
		t.Fatalf("custom phrase: status=%q rule=%q reason=%q", v.Status, v.Rule, v.Reason)
	}
	if len(v.Signals) == 0 || v.Signals[0] != "moonlight from mars" {
		t.Fatalf("expected the matched signal recorded, got %v", v.Signals)
	}
}

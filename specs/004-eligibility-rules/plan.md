# Implementation Plan: Eligibility Rules

**Branch**: `004-eligibility-rules` | **Date**: 2026-09-21 | **Spec**: [spec.md](./spec.md)

## Summary

A deterministic, fully configurable eligibility filter that runs inside the ingest
loop after scoring and before drafting: worldwide/global-remote/hire-anywhere and
sponsored-relocation roles pass (the latter flagged for relocation highlighting in
the cover letter), country-bounded remote roles and roles requiring existing local
work rights are vetoed and archived with a human-readable reason — producing no
draft and no alert.

## Technical Context

**Language/Version**: Go 1.25 (backend), TypeScript/Next.js 15 (frontend)

**Primary Dependencies**: chi, sqlx, `modernc.org/sqlite`, `gopkg.in/yaml.v3`; no new dependencies

**Storage**: SQLite `data/jobs.db` (four additive `jobs` columns); rules in `data/settings.yaml` under an `eligibility:` block

**Testing**: `go test ./...` (table-driven `services_test`, `db_test`, api smoke); `npm run lint && npm run typecheck && npm run build`

**Target Platform**: same as today — linux/WSL localhost, `:8000` API + `:3000` UI

**Constraints**: no per-posting network call (free, offline, reproducible); no fake verdicts — every veto names its rule and matched phrases; local data truth (the backend decides and persists, the dashboard renders)

**Scale/Scope**: single operator; ~250 lines of new backend logic + one settings section, one board toggle, one detail panel

## Constitution Check

| Principle | Status |
|---|---|
| I. Self-hosted personal control center | PASS — rules live in the operator's own `settings.yaml`, no service dependency |
| II. Honest automation | PASS — vetoed roles are archived and labelled, never presented as applied or as a match; no draft is claimed that was not written |
| III. Local data truth | PASS — verdict stored on the job; the UI never recomputes a verdict |
| IV. Privacy | PASS — evaluation is local string matching; no job text leaves the machine |
| V. Operational clarity | PASS — one line in the run summary, one badge on the card, one reason on the role |

Post-design: unchanged — PASS.

## Key decisions

1. **Rules live in `data/settings.yaml`** as a nested `eligibility:` block, reusing the
   existing load/validate/atomic-save plumbing and `GET/PUT /api/settings`. A separate
   file would duplicate that path for no benefit.
2. **Deterministic phrase rules, not an LLM call.** Free, offline, reproducible, and
   every verdict carries evidence. Vetoes beat passes; no signal ⇒ `unknown` ⇒ pass
   (never veto on a guess), which the operator can disable.
3. **Verdict persisted on the job** (`eligibility`, `eligibility_rule`,
   `eligibility_reason`, `eligibility_signals`) via the existing additive-column
   migration helper.
4. **Veto means Archived (`rejected`)**, which already maps to the Archived column —
   no new pipeline state, and a wrong verdict stays reversible.
5. **`HighlightsRelocation` is a pass with a drafting hint**, and the hint only uses
   `work_authorization`/`relocation` facts the operator actually wrote in
   `project_bank.yaml`, so nothing is invented.
6. **Re-check is safe**: it only flips jobs whose own stored verdict is a veto; roles
   the operator moved by hand are left alone.

## Structure

```text
backend/internal/
├── services/eligibility.go        # rules model, defaults, merge, EvaluateEligibility
├── services/eligibility_test.go
├── services/settings.go           # Eligibility field + validation
├── services/ingest.go             # veto → archived, skip draft/notify, summary count
├── services/drafter.go            # relocation-readiness hint
├── api/eligibility.go             # preview + reapply handlers
├── api/sources.go                 # settings payload includes eligibility + defaults
├── api/jobs.go                    # ?eligibility= filter
├── api/openapi.yaml
├── db/schema.sql, db/db.go, db/store.go, db/store_dashboard.go
└── models/models.go

frontend/src/
├── app/settings/page.tsx          # rules editor, phrase chips, JD tester, re-check
├── app/page.tsx                   # hide-vetoed toggle
├── components/JobCard.tsx         # veto badge
├── app/jobs/[id]/page.tsx         # eligibility panel
└── lib/api.ts                     # types + preview/reapply calls

specs/004-eligibility-rules/{spec,plan,tasks}.md
```

# Tasks: Eligibility Rules

**Input**: Design documents from `/specs/004-eligibility-rules/`

**Prerequisites**: plan.md, spec.md

**Tests**: Required — table-driven Go tests for every rule in the stated policy (SC-003).

## Phase 1: Setup

- [ ] T001 Spec + plan artifacts, feature pointer in `.specify/feature.json`
- [ ] T002 [P] `data/settings.yaml.example` + `make setup` seeding it

## Phase 2: Rule engine (services)

- [ ] T003 [P] `EligibilityRules` model + defaults + merge in `backend/internal/services/eligibility.go`
- [ ] T004 [P] `EvaluateEligibility` (pure, deterministic) with rule precedence and human reasons
- [ ] T005 Validation of rules/phrases/country codes in `services/settings.go`

## Phase 3: Persistence (US1/US2 truth)

- [ ] T006 `jobs` columns (`eligibility`, `eligibility_rule`, `eligibility_reason`, `eligibility_signals`) in `schema.sql`
- [ ] T007 Additive migration via `addColumnIfMissing` in `internal/db/db.go`
- [ ] T008 `models.Job`/`JobListItem` fields + all job SELECTs + `Store.SetJobEligibility`

## Phase 4: US1 — Veto before effort

- [ ] T009 Evaluate in the ingest loop; veto → status `rejected`, no draft, no notify
- [ ] T010 `IngestStatus.Vetoed` + run summary line

## Phase 5: US2 — Pass + relocation highlight

- [ ] T011 `HighlightsRelocation` → drafter prompt hint (`drafter.go`)
- [ ] T012 Optional `candidate.work_authorization` / `candidate.relocation` facts in `project_bank.yaml(.example)`

## Phase 6: US3 — Operator configuration

- [ ] T013 `GET/PUT /api/settings` eligibility block + defaults
- [ ] T014 `POST /api/eligibility/preview` (no writes)
- [ ] T015 `POST /api/eligibility/reapply` (stored jobs, manual moves preserved)
- [ ] T016 `?eligibility=veto|pass` filter on `GET /api/jobs`
- [ ] T017 `openapi.yaml`: `EligibilityRules`, `EligibilityVerdict`, new paths, Job/Settings schemas
- [ ] T018 Settings page section: rule toggles, country chips, phrase editors, JD tester, re-check action
- [ ] T019 Board: "Hide vetoed" toggle (localStorage), hidden count, veto badge with reason
- [ ] T020 Job detail: Eligibility panel (verdict, rule, reason, signals, re-check)

## Phase 7: Docs

- [ ] T021 README eligibility section + troubleshooting entry
- [ ] T022 `data/README.md`, `docs/architecture.md`, `CHANGELOG.md`

## Phase 8: Verification

- [ ] T023 `services/eligibility_test.go` — every pass/veto case, exemption, unknown, disabled, custom phrases
- [ ] T024 Settings defaults/merge/validation tests
- [ ] T025 `migrate_test.go` legacy DB gains columns; verdict round-trip
- [ ] T026 api smoke: settings block + preview verdicts
- [ ] T027 `go build ./... && go vet ./... && go test ./...`; `npm run lint && typecheck && build`

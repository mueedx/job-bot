# Implementation Plan: Control Center Dashboard

**Branch**: `001-control-center` | **Date**: 2026-09-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-control-center/spec.md`

## Summary

Deliver a localhost control-center UI so Mueed can scan a job pipeline board, review JD/match/cover letter, switch resume track, discard or attempt approve-and-submit (with honest not-ready messaging), and view analytics. Extend the existing Go API just enough to serve match + application drafts; build the dashboard in Next.js 15.

## Technical Context

**Language/Version**: Go 1.25 (backend); TypeScript 5 / Node 20+ (frontend)

**Primary Dependencies**: chi, sqlx, modernc.org/sqlite (backend); Next.js 15 App Router, Tailwind CSS, Lucide Icons (frontend)

**Storage**: SQLite `data/jobs.db` (jobs, matches, applications, submission_logs)

**Testing**: `go test ./internal/api/`; manual browser verification via quickstart

**Target Platform**: Linux/WSL localhost (`:8000` API, `:3000` UI)

**Project Type**: Web application (Go API + Next.js dashboard)

**Performance Goals**: Board and detail loads feel instant for <500 local jobs

**Constraints**: Constitution — no fake submit success; local data truth; no multi-tenant auth

**Scale/Scope**: Single operator; 3 pages (board, detail, stats); thin API enrichment

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status |
|---|---|
| I. Self-hosted personal control center | PASS — localhost only, no auth wall |
| II. Honest automation | PASS — Approve & Submit surfaces 501/not-ready; no fake sent |
| III. Local data truth | PASS — persist via Go API; seed only via explicit endpoint |
| IV. Privacy | PASS — dashboard talks only to local API |
| V. Operational clarity | PASS — Kanban + detail + analytics; no marketing landing |

Post-design: unchanged — PASS.

## Project Structure

### Documentation (this feature)

```text
specs/001-control-center/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/server/main.go
├── internal/api/          # handlers + openapi
├── internal/db/           # store + schema
├── internal/models/
├── data/
└── resumes/

frontend/
├── src/app/
│   ├── layout.tsx
│   ├── page.tsx           # Kanban
│   ├── jobs/[id]/page.tsx
│   └── stats/page.tsx
├── src/components/
├── src/lib/api.ts
└── package.json
```

**Structure Decision**: Existing `backend/` + `frontend/` split; extend Go packages; replace frontend stub with Next.js App Router under `frontend/src/`.

## Complexity Tracking

> None — no constitution violations.

# Tasks: Control Center Dashboard

**Input**: Design documents from `/specs/001-control-center/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Optional smoke via existing Go tests + quickstart manual checks (no separate test suite required by spec).

## Phase 1: Setup

- [x] T001 Replace `frontend/` stub with Next.js 15 App Router + Tailwind + TypeScript
- [x] T002 [P] Add `frontend/.env.local.example` with `NEXT_PUBLIC_API_URL=http://localhost:8000`
- [x] T003 [P] Add `frontend/src/lib/api.ts` typed client for jobs/detail/application/stats/seed

## Phase 2: Foundational (API)

- [x] T004 Extend store: GetMatch, GetApplication, UpsertApplication, ListJobsEnriched, SeedDemo in `backend/internal/db/store.go`
- [x] T005 [P] Add detail/application/discard/approve/seed handlers in `backend/internal/api/`
- [x] T006 Wire routes in `backend/internal/api/router.go`; extend stats with `daily_limit`
- [x] T007 Update `backend/internal/api/openapi.yaml` + Go smoke test for new endpoints
- [x] T008 [P] Extend models in `backend/internal/models/models.go` for detail DTOs

**Checkpoint**: `curl` detail/seed/stats works with Go server

## Phase 3: User Story 1 — Pipeline board (P1) 🎯 MVP

**Goal**: Scan columns and open a role

- [x] T009 [US1] App shell + nav in `frontend/src/app/layout.tsx` and `AppNav.tsx`
- [x] T010 [US1] `KanbanColumn.tsx` + `JobCard.tsx` with status→column mapping
- [x] T011 [US1] Kanban page `frontend/src/app/page.tsx` fetching enriched jobs
- [x] T012 [US1] Empty/error states when API down

**Checkpoint**: Seeded board navigates to `/jobs/[id]`

## Phase 4: User Story 2 — Review & edit draft (P1)

- [x] T013 [US2] Job detail page loads `/detail` in `frontend/src/app/jobs/[id]/page.tsx`
- [x] T014 [P] [US2] `CoverLetterEditor.tsx` with save via PUT application
- [x] T015 [US2] Match breakdown + track badge; switch track control

**Checkpoint**: Save letter + switch track persists across reload

## Phase 5: User Story 3 — Approve / discard (P1)

- [x] T016 [US3] `ActionBar.tsx` Discard + Approve & Submit
- [x] T017 [US3] Honest error toast/banner on 501/not-ready; discard refreshes board column

**Checkpoint**: Discard archives; approve never fakes sent

## Phase 6: User Story 4 — Analytics (P2)

- [x] T018 [US4] `frontend/src/app/stats/page.tsx` from `/api/stats` including daily_limit

## Phase 7: Polish

- [x] T019 Visual polish (IBM Plex, teal accent, grid wash) in `globals.css` / layout
- [x] T020 Update root `README.md` with dual-server quickstart
- [x] T021 Run quickstart.md verification checklist

## Dependencies

- Setup → Foundational → US1 → US2 → US3 → US4 → Polish
- US1 is MVP; US2/US3 can follow immediately after API foundation

## Implementation Strategy

1. Finish API foundation first (blocks all UI stories).
2. Ship board (US1), then detail editor (US2), then actions (US3), then stats (US4).
3. Stop if Approve ever implies success without backend confirmation.

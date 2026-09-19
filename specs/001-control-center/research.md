# Research: Control Center Dashboard

## Decision 1 — Extend Go API vs mock-only UI

**Decision**: Extend Go with detail/application/discard/approve/enrich/seed endpoints.

**Rationale**: Constitution III requires backend-authoritative status and drafts. Mock-only UI would invent state.

**Alternatives considered**: Frontend-only mocks (rejected — fake success risk); wait for Phase 2–3 scrapers/drafter (rejected — blocks Stage 1 calibration UI).

## Decision 2 — Next.js 15 App Router + Tailwind + Lucide

**Decision**: Follow product roadmap §7 for dashboard stack.

**Rationale**: Already locked in job-agent-spec; operator knows Next.js.

**Alternatives considered**: SPA Vite+React (extra deviation); server-rendered Go templates (poorer editor UX).

## Decision 3 — Approve & Submit still returns not-implemented

**Decision**: Approve may set status `approved` then attempt submit; if submit unavailable, return explicit error; UI toasts/banners that message. Do not set `applied` unless submit succeeds.

**Rationale**: Constitution II.

## Decision 4 — Kanban status mapping

**Decision**:

| Column | Statuses |
|---|---|
| Discovered | `discovered` |
| High Match (Ready) | `scored`, `queued`, `approved` |
| Applied | `applied` |
| Interview | `interview` |
| Archived | `rejected` |

**Rationale**: Matches locked product assumptions in spec.

## Decision 5 — Visual system

**Decision**: Light operational UI; IBM Plex Sans + Mono; teal accent `#0f766e`; cool gray wash.

**Rationale**: Dashboard exception to marketing-hero rules; avoid purple/cream/broadsheet clichés.

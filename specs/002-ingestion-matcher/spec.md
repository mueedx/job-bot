# Feature Specification: Ingestion & Matcher

**Feature Branch**: `002-ingestion-matcher`  
**Created**: 2026-09-07  
**Status**: Draft  
**Input**: Discover jobs from public boards, score/route to resume tracks, fill pipeline without demo seed.

## User Scenarios & Testing

### User Story 1 - Discover roles from public boards (Priority: P1)

Mueed triggers ingestion and new roles from curated company boards and public job feeds appear in the pipeline.

**Independent Test**: Run ingest once; at least one new role appears (or soft-fail logged per source) without demo seed.

**Acceptance Scenarios**:
1. **Given** curated company lists, **When** ingest runs, **Then** public board sources are queried and unique URLs are stored.
2. **Given** a source fails, **When** ingest continues, **Then** other sources still complete and errors are recorded for status.

### User Story 2 - Score and route (Priority: P1)

Each new role gets a resume track and match score so the board can show High Match (Ready).

**Independent Test**: A fullstack-flavored JD routes fullstack; a Solidity JD routes blockchain; Web3+customer AI routes fde.

**Acceptance Scenarios**:
1. **Given** a new job, **When** matcher runs, **Then** a match row exists with track, score, skills, reasons.
2. **Given** score ≥ notify threshold, **When** scoring completes, **Then** status becomes scored or queued (not applied).

## Requirements

- **FR-001**: System MUST ingest from public Greenhouse, Lever, Ashby boards plus RemoteOK and a crypto/web3 public feed.
- **FR-002**: System MUST dedupe by URL.
- **FR-003**: System MUST assign track fullstack|blockchain|fde per keyword rules with documented tie-break.
- **FR-004**: System MUST compute a 0–1 score with compensation/remote heuristics; missing salary MUST NOT auto-fail.
- **FR-005**: Operator MUST be able to trigger ingest and view last-run status.
- **FR-006**: LinkedIn/Wellfound cookie scraping MUST NOT be required for this feature.

## Success Criteria

- **SC-001**: Ingest run completes with per-source success/error summary.
- **SC-002**: New jobs appear on the control-center board without demo seed when sources return data.
- **SC-003**: Tie-break cases match the product track rules in tests.

## Assumptions

- Public APIs only; operator machine has network access.
- No auto-submit in this feature.

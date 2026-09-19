# Feature Specification: Drafter & Telegram

**Feature Branch**: `003-drafter-telegram`  
**Created**: 2026-09-07  
**Status**: Draft  
**Input**: Draft cover letters from project bank; Telegram alerts and commands; honest approve.

## User Scenarios & Testing

### User Story 1 - Draft cover letter (Priority: P1)

High-scoring roles get a cover letter grounded only in Mueed's project bank.

**Acceptance Scenarios**:
1. **Given** OpenAI key set and score ≥ threshold, **When** draft runs, **Then** application cover letter is persisted.
2. **Given** no API key, **When** ingest/match runs, **Then** system skips draft without failing ingest.

### User Story 2 - Telegram control (Priority: P1)

Mueed receives high-match alerts and can discard or attempt approve from Telegram; commands show stats/today/status.

**Acceptance Scenarios**:
1. **Given** bot configured, **When** high match appears, **Then** alert with approve/discard is sent.
2. **Given** approve pressed, **When** submitter missing, **Then** reply states not sent / Phase 5.
3. **Given** `/stats` `/today` `/status`, **When** sent, **Then** useful summaries return.

## Requirements

- **FR-001**: Drafter MUST use only configured project-bank facts.
- **FR-002**: Draft trigger by threshold and manual `POST /api/jobs/{id}/draft`.
- **FR-003**: Telegram optional when token+chat set.
- **FR-004**: Approve MUST NOT claim application sent if submitter unavailable.

## Success Criteria

- **SC-001**: Detail UI shows drafted letter after draft run with key present.
- **SC-002**: 100% of approve attempts without submitter show honest failure message.

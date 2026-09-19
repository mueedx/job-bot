# Feature Specification: Control Center Dashboard

**Feature Branch**: `001-control-center`

**Created**: 2026-09-06

**Status**: Draft

**Input**: User description: "As Mueed reviewing daily job matches on my laptop, I need a localhost control center where I can scan a pipeline board, open a role to read the JD and match rationale, edit the cover letter, switch resume track, discard weak fits, and attempt approve-and-submit. I also need a simple analytics view of how many roles are discovered, applied, and in interview so I can pace Stage 1 calibration. When submission is not ready yet, the product must say so clearly instead of implying success."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scan the pipeline board (Priority: P1)

Mueed opens the control center and sees open roles grouped into pipeline columns so he can decide what to review next without opening every listing.

**Why this priority**: Without a scannable board, Stage 1 calibration has no daily cockpit; everything else hangs off this view.

**Independent Test**: With sample roles in several pipeline states, open the board and confirm each role appears in the correct column with enough identity (title, company, match signal when present) to choose one.

**Acceptance Scenarios**:

1. **Given** roles exist in multiple pipeline states, **When** Mueed opens the control center home view, **Then** he sees columns Discovered, High Match (Ready), Applied, Interview, and Archived.
2. **Given** a role is visible on the board, **When** he selects it, **Then** he navigates to that role’s detail view.
3. **Given** a role has a match score and resume track, **When** it appears on the board, **Then** those signals are visible on the card without opening the detail view.

---

### User Story 2 - Review and edit a role draft (Priority: P1)

Mueed opens a role, reads the job description and match rationale, edits the cover letter, and can switch which resume track will be used.

**Why this priority**: Stage 1 exists to verify drafts before any send; editing and track choice are the core human loop.

**Independent Test**: Open one role with match and draft text, change the cover letter, save, switch track, reload, and confirm persisted values.

**Acceptance Scenarios**:

1. **Given** a role with a job description, **When** Mueed opens its detail view, **Then** he can read the full description.
2. **Given** a role with match data, **When** he views detail, **Then** he sees score, rationale, matched skills, missing skills, and the selected resume track.
3. **Given** an editable cover letter, **When** he changes the text and saves, **Then** the updated letter is retained after leaving and returning to the role.
4. **Given** tracks fullstack, blockchain, and fde, **When** he switches resume track, **Then** the new track is shown and retained.

---

### User Story 3 - Decide: approve, discard, or defer (Priority: P1)

Mueed discards weak fits or attempts approve-and-submit. If submission cannot complete yet, the product says so without claiming success.

**Why this priority**: Honest Stage 1 decisions are a constitutional requirement; fake “sent” states are forbidden.

**Independent Test**: Discard moves a role to Archived; approve when submission is unavailable shows a clear not-ready message and does not mark the role as successfully applied solely in the UI.

**Acceptance Scenarios**:

1. **Given** a role under review, **When** Mueed discards it, **Then** it appears under Archived on the board.
2. **Given** submission is not available yet, **When** he chooses Approve & Submit, **Then** he sees an explicit message that submission is not ready / not implemented, and the product does not claim the application was sent.
3. **Given** a role was discarded, **When** he returns to the board, **Then** it is no longer in Discovered or High Match (Ready).

---

### User Story 4 - Pace with analytics (Priority: P2)

Mueed opens analytics to see how many roles are discovered, applied, and in interview (plus a status breakdown) and a configured daily quota reminder.

**Why this priority**: Helps Stage 1 pacing but is secondary to reviewing individual roles.

**Independent Test**: With known counts in the pipeline, open analytics and verify totals and breakdown match; quota limit is visible as configured guidance.

**Acceptance Scenarios**:

1. **Given** roles in various statuses, **When** Mueed opens analytics, **Then** he sees totals for discovered/applied/interview (or equivalent pipeline counts) and a by-status breakdown.
2. **Given** a configured daily application limit, **When** he views analytics, **Then** that limit is shown as guidance even if live send counters are not yet available.

---

### Edge Cases

- What happens when the local backend is unreachable? The control center shows a clear error/empty state and does not invent jobs.
- What happens when a role has no match or no cover letter yet? Detail still opens; missing sections are empty or clearly marked, not fabricated.
- What happens on duplicate save or empty cover letter? Save either accepts empty draft or shows a validation message; it must not report success if persistence failed.
- What happens if Approve & Submit partially updates status but cannot submit? User must still see that submission did not complete successfully.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST present a pipeline board with columns Discovered, High Match (Ready), Applied, Interview, and Archived.
- **FR-002**: System MUST place each role into exactly one board column based on its pipeline status.
- **FR-003**: Users MUST be able to open a role detail view from the board.
- **FR-004**: Role detail MUST show job description text when available.
- **FR-005**: Role detail MUST show match score, rationale, matched skills, missing skills, and resume track when match data exists.
- **FR-006**: Users MUST be able to edit and persist a cover letter for a role.
- **FR-007**: Users MUST be able to switch resume track among fullstack, blockchain, and fde and persist the choice.
- **FR-008**: Users MUST be able to discard a role so it appears in Archived.
- **FR-009**: Users MUST be able to attempt Approve & Submit from role detail.
- **FR-010**: When submission cannot be completed, the system MUST communicate failure/not-ready explicitly and MUST NOT claim the application was sent.
- **FR-011**: System MUST provide an analytics view with applied/interview-oriented counts and a status breakdown.
- **FR-012**: Analytics MUST show the configured daily application limit as guidance when live send counters are unavailable.
- **FR-013**: Board cards SHOULD surface match score and track when present so scanning does not require opening every role.
- **FR-014**: System MUST load pipeline and draft data from the authoritative local backend; client-only fake success states are prohibited.

### Key Entities *(include if feature involves data)*

- **Role (Job)**: A posting under review; has title, company, description, location/compensation signals, pipeline status, and link identity.
- **Match**: Fit assessment for a role — score, rationale, matched/missing skills, resume track.
- **Application draft**: Cover letter text and selected resume path/track for a role; submission outcome when attempted.
- **Pipeline status**: Lifecycle label that maps to board columns (including archived/discarded).
- **Analytics summary**: Aggregated counts by status and configured daily quota guidance.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operator can open the control center and identify which role to review next from the board in under 30 seconds when sample data is present.
- **SC-002**: Operator can complete a review loop (open role → edit cover letter → save → switch track) in under 3 minutes for a single role.
- **SC-003**: In 100% of attempts where submission is unavailable, the UI communicates not-ready/failure and never shows a false “sent” confirmation.
- **SC-004**: Discarded roles appear in Archived on the next board view without manual refresh hacks (normal navigation/reload is enough).
- **SC-005**: Analytics totals for applied and interview match the operator’s known pipeline counts for a seeded dataset.
- **SC-006**: With the backend stopped, the operator sees a clear failure/empty state rather than fabricated roles.

## Assumptions

- Single operator on localhost; no login wall for Phase 4.
- Board column mapping follows the product constitution (Discovered; High Match Ready for scored/queued/approved-class statuses; Applied; Interview; Archived for rejected/discarded).
- Resume tracks are exactly fullstack, blockchain, and fde.
- Submission engine may be incomplete; honesty about that state is required.
- Daily quota shown as configured limit (default guidance such as 5–10/day product intent) until real send counters exist.
- Sample/seed data may be used for local demonstration of the board.
- Scrapers, LLM drafting, Telegram, and real browser submission are out of scope for this feature.

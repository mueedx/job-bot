# Feature Specification: Eligibility Rules

**Feature Branch**: `004-eligibility-rules`

**Created**: 2026-09-21

**Status**: Draft

**Input**: User description: "Roles must pass an eligibility filter before any drafting effort is spent. Worldwide / global remote / hire-anywhere (EOR or contractor) roles pass. On-site or hybrid roles pass when the posting offers visa sponsorship and relocation assistance, and the cover letter should highlight relocation readiness. Country-bounded remote roles ('Remote — US only', 'Remote — UK only', 'Must reside in Poland/Germany') are an automatic veto, and so are roles that require existing local citizenship or a work permit. The operator defines these constraints and every part of the policy is configurable."

## User Scenarios & Testing

### User Story 1 - Veto roles I cannot take (Priority: P1)

Roles that are locked to a country I am not authorized to work in, or that demand existing local citizenship or a work permit, never reach my review queue and never cost a drafting request.

**Why this priority**: The whole point of the filter is to stop wasting review and drafting effort on roles that are legally out of reach.

**Independent Test**: Feed a posting whose location says "Remote — US only" and one whose text says "must be authorized to work in the US without sponsorship"; both are vetoed with a plain-language reason, appear as Archived rather than High Match, and produce no drafted application.

**Acceptance Scenarios**:

1. **Given** eligibility rules are enabled, **When** a posting is bounded to a country I am not work-authorized in, **Then** it is vetoed, its reason names the country, and it is not queued as a high match.
2. **Given** eligibility rules are enabled, **When** a posting requires existing local citizenship, a local work permit, or explicitly refuses sponsorship, **Then** it is vetoed with that reason.
3. **Given** a vetoed posting, **When** an ingest run continues, **Then** no cover letter or application record is created for it and no alert is sent about it.
4. **Given** the run completes, **When** I read the run summary, **Then** the number of vetoed postings is reported alongside inserted and scored counts.

---

### User Story 2 - Pass the roles that are actually reachable (Priority: P1)

Roles that are genuinely open to me pass, including global remote and hire-anywhere roles, and on-site or hybrid roles that sponsor a visa and help with relocation.

**Why this priority**: A filter that only rejects is useless; it must not hide the work I can take.

**Independent Test**: Postings for "Worldwide remote (any timezone)", "Hire anywhere via employer of record", and "On-site Dublin — visa sponsorship and relocation package provided" all pass; the sponsored one is flagged for relocation highlighting.

**Acceptance Scenarios**:

1. **Given** a posting that is worldwide, global remote, or explicitly hire-anywhere (EOR/contractor), **When** eligibility is evaluated, **Then** it passes.
2. **Given** an on-site or hybrid posting that offers visa sponsorship together with relocation assistance, **When** eligibility is evaluated, **Then** it passes and is marked as relocation-ready.
3. **Given** a posting with no location or eligibility signal at all, **When** eligibility is evaluated, **Then** it is not vetoed on a guess — it passes through as unknown.

---

### User Story 3 - Define and tune the constraints myself (Priority: P1)

I decide which rules are on, which countries I already have work rights in, and which phrases count as a pass or a veto — and I can test a posting against the rules before and after a search.

**Why this priority**: The policy is personal (my citizenship, my permit, my markets), so it must be configuration, not hardcoded behaviour.

**Independent Test**: Turn the country-bounded rule off, re-check existing jobs, and previously vetoed roles return to the board; add my own phrase and see it change a verdict in the tester.

**Acceptance Scenarios**:

1. **Given** the settings page, **When** I change any eligibility rule, country list, or phrase list and save, **Then** the change is persisted, survives a restart, and is used by the next search.
2. **Given** a country list naming the countries where I already hold work rights, **When** a remote role is bounded to one of those countries while the exemption is on, **Then** it passes instead of being vetoed.
3. **Given** the settings page tester, **When** I paste a posting's title, location, and description, **Then** I see the verdict, the rule that decided it, and the phrases that matched — without a search running and without storing anything.
4. **Given** rules I just changed, **When** I ask for existing jobs to be re-checked, **Then** stored jobs are re-evaluated without scraping again, and roles I moved by hand are left where I put them.
5. **Given** eligibility rules are switched off entirely, **When** a search runs, **Then** nothing is vetoed and the board behaves as before.

---

### Edge Cases

- A posting mentions both a pass signal and a veto signal (e.g. "remote worldwide, but you must be authorized to work in Canada"): a veto always wins.
- A posting is bounded to a country I am work-authorized in, but relocation sponsorship is absent — it passes on the exemption; only the country-bound veto depends on that list.
- My work-authorized country list is empty: every country-bounded remote role is vetoed, and the settings page says so plainly rather than failing silently.
- Phrase lists are blanked by hand or country codes are typo'd: the invalid configuration is reported instead of quietly matching nothing.
- A vetoed role is later edited by hand or the rules change: it can be re-checked and, when it now passes, it returns to the pipeline rather than staying archived.
- Keywords are matched case-insensitively, including in titles, locations, and descriptions.

## Requirements

- **FR-001**: The system MUST evaluate every newly ingested posting against an eligibility policy before any cover letter is drafted.
- **FR-002**: The policy MUST treat worldwide / global remote / hire-anywhere (EOR or contractor) postings as passing.
- **FR-003**: The policy MUST treat on-site or hybrid postings that offer visa sponsorship together with relocation assistance as passing, and MUST mark them as relocation-ready so drafting can highlight relocation readiness.
- **FR-004**: The policy MUST veto remote postings bounded to a country the operator is not work-authorized in, and postings that require existing local citizenship or a work permit, naming the deciding rule and matched phrases in a human-readable reason.
- **FR-005**: A veto MUST NOT be a guess: postings with no eligibility signal MUST pass through as unknown unless the operator disables that behaviour.
- **FR-006**: Vetoed postings MUST NOT receive a drafted cover letter, an application record, or a notification, and MUST NOT be presented as high match.
- **FR-007**: Every part of the policy MUST be operator-configurable — master enable, each individual rule, the work-authorized country list with its own exemption switch, and the phrase lists that drive pass and veto decisions — persisted with the rest of the dashboard settings and validated with clear errors.
- **FR-008**: The operator MUST be able to test a posting against the current policy without running a search and without persisting anything.
- **FR-009**: The operator MUST be able to re-evaluate already-stored postings after changing the policy, without re-scraping, and without undoing manual pipeline moves.
- **FR-010**: The verdict, its rule, its reason, and its matched phrases MUST be stored with the posting and shown on the board and the role detail view so the dashboard never recomputes or invents a verdict.
- **FR-011**: The ingest run summary MUST report how many postings were vetoed.

## Success Criteria

- **SC-001**: 100% of postings that require existing local work rights and 100% of country-bounded remote postings outside the operator's authorized countries are vetoed, with a reason a human can check against the posting text.
- **SC-002**: 0 drafted applications and 0 notifications are produced for vetoed postings.
- **SC-003**: The three pass cases and the two veto cases from the stated policy are covered by automated tests.
- **SC-004**: Changing a rule and re-checking stored jobs changes verdicts on the existing board without a new search.

## Assumptions

- Eligibility is decided from the posting text the operator already stores (title, location, description) using deterministic phrase rules — no per-posting external call, so it is free, offline, and reproducible.
- "Work-authorized" country codes are the operator's own statement of where they already hold citizenship or a permit; the system does not verify legal status.
- Vetoed postings are archived rather than deleted, so a wrong verdict stays reviewable and reversible.
- Phrase lists ship with sensible defaults so the stated policy works on first run, and can be replaced entirely.

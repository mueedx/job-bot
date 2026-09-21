# Tech Stack & Design Rationale

This document explains **what** the project is built with and, more
importantly, **why** every choice was made: the database, the algorithms, the
loops, and the principles that shaped the code. It complements
[architecture.md](architecture.md), which describes *how* the code is
organised, and [job-agent-spec.md](job-agent-spec.md), which is the original
planning context.

---

## 1. The stack at a glance

| Layer | Choice | Why this, not the obvious alternative |
|---|---|---|
| Backend language | **Go (stdlib only)** | One static binary, no runtime to install on a home server, and `net/http` is genuinely enough for ~20 endpoints. FastAPI/Django would pull a language runtime, a virtualenv and a framework for the same result. Go's compile-time checks also fit a project that runs unattended: most failure modes are caught before the binary is built. |
| HTTP layer | `net/http` + a hand-written router | No framework magic. Every handler is a plain function, every route is one line in `router.go`, and the OpenAPI test fails the build if a route is undocumented. A framework would buy routing sugar and cost transparency. |
| Database | **SQLite** (single file, `data/jobs.db`) | See §2 — this is the biggest single decision. |
| Migrations | Hand-rolled, sequential, additive | See §3. |
| Frontend | **Next.js 15 App Router + Tailwind** | The dashboard is a small, stateful CRUD surface. App Router gives client components exactly where interactivity is needed (drag-and-drop board, editors) and server rendering everywhere else. Tailwind keeps a single-person project free of a design-system layer to maintain. |
| LLM | Optional, endpoint-agnostic, key-gated | Every AI feature is a **supplement** (§6). With no key, the pipeline still ingests, scores, and tracks — it just drafts nothing. |
| Config | **YAML files in `data/`** | The operator owns `settings.yaml`, `target_companies.yaml` and `project_bank.yaml`. YAML round-trips comments and is diff-friendly, which JSON is not; a database would hide configuration from `git` and from `cat`. |

The recurring theme: **local-first, boring, inspectable.** No cloud service,
no account, no telemetry. Everything that matters lives in one directory you
can back up with `cp -r`.

---

## 2. Why SQLite

The database decision was made on four grounds, in order of importance:

1. **Deployment shape.** This is a single-user, single-process application on
   a laptop or home server. A network database (Postgres, MySQL) adds a
   daemon, credentials, backups, and a failure domain for zero benefit at a
   write rate of a few hundred rows per ingest run. SQLite's whole value
   proposition is exactly this shape: one file, one process, zero config.
2. **Backup equals `cp`.** The entire pipeline state — jobs, matches,
   applications, settings history — is one file in `data/`. Docker Compose
   bind-mounts that one directory, so a local run and a container share state
   with no volume plumbing.
3. **Write concurrency profile.** Ingest writes in bursts (one run every few
   hours), the API reads constantly, and nothing else writes. SQLite's
   single-writer model is a *feature* here: it serialises ingest bursts
   safely without locks, retries, or connection pools. WAL mode keeps readers
   unblocked while ingest writes.
4. **The data is small and relational.** Jobs, matches, applications are
   naturally relational with foreign keys; a document store would push
   consistency work into application code. Ten thousand postings fit in a few
   megabytes.

The honest limits: no multi-user writes, and no remote access to the DB. Both
are out of scope by design. If remote multi-client ever matters, the API —
not the DB — is the boundary, and the store interface is the seam to swap.

### How the DB is used

- One `*sql.DB` with `PRAGMA foreign_keys = ON` and WAL enabled at open.
- A single `Store` type owns every query (no ORM, no query builder). SQL is
  written once, visible, and testable — the store tests run migrations
  against a temp file and exercise real SQL, not mocks.
- Writes are idempotent by **natural key**: `(source, source_id)` and a
  unique `url`. Re-running an ingest over the same postings updates rather
  than duplicates. This is what makes "run ingest whenever you feel like it"
  safe.

---

## 3. Migrations

Migrations are sequential and additive, applied at startup by `db.go`, and
covered by a migration test that asserts every version boundary can be
crossed from an empty file and from each prior version.

Why hand-rolled instead of a migration library: there are fewer than ten
migrations, they must work from a fresh checkout *and* from a database that
predates the feature, and the whole mechanism is ~80 lines. A library would
add a dependency to solve a problem this project does not have yet. The rule
that keeps it safe: **migrations only add columns/tables — they never rewrite
or drop.** An older database that skips several versions in one startup
arrives at the same schema as a fresh one.

---

## 4. The ingest loop

`services/ingest.go` runs one pass over all enabled sources:

1. **Load targets** — the registry of source specs intersected with the
   operator's settings (toggles, regions, eligibility rules).
2. **Fetch** — each enabled scraper runs independently; one source failing
   logs a warning and the run continues. Per-source failures never abort the
   batch, because a broken board should not stop the three healthy ones from
   delivering postings.
3. **Deduplicate** — upsert by natural key; already-seen postings are
   refreshed, not re-created.
4. **Score** — the matcher (§5) scores every new posting.
5. **Eligibility** — the rule engine (§7) decides pass/veto *before* any
   drafting. Vetoed postings are archived immediately.
6. **Draft** — only for postings that survived scoring and eligibility, only
   when a key is configured. This ordering is deliberate: the LLM is the most
   expensive and slowest step in the loop, so it is the *last* step, guarded
   by two cheap, deterministic filters.
7. **Summarise** — one Telegram/dashboard summary with counts, including how
   many postings were vetoed by eligibility and therefore cost nothing.

The loop is triggered manually or on a schedule; it is not a long-lived
daemon. State between runs lives entirely in the database, which is why any
run can crash at any point without corrupting anything: each step's writes
are committed before the next step starts.

---

## 5. The matcher: scoring algorithm

Matching is two layers, by design:

**Layer 1 — deterministic heuristics (always on).** Each posting is scored
0.0–1.0 from compensation fit, remote/relocation alignment, track signals
from title and description, and a small keyword vocabulary per track. Every
contributing signal is recorded in `score_reasons` in plain language. The
principle: **a score must be explainable.** A blank or surprising number is a
bug, not a black box.

**Layer 2 — AI supplement (optional).** Resumes are analysed once per file,
cached by content hash, to extract track, skills and keywords. The matcher
adds a *bounded* keyword boost — bounded so a misbehaving model can never
dominate the heuristic floor. Without a key the system loses nothing; with
one it gets better at routing, never worse than the heuristics.

**Track routing.** Each posting is assigned one of three master resume tracks
(`fullstack`, `blockchain`, `fde`) by keyword evidence, with fullstack as the
default. The operator can override the track per job on the detail page; the
override wins over the classifier because the human is the authority on their
own CV.

---

## 6. The drafter: grounding over generation

The drafter is key-gated and grounded in `data/project_bank.yaml` — the
operator's own, hand-editable project bank. The LLM composes from real,
verified bullets; it is never asked to invent experience. The bank, not the
model, is the source of truth about the candidate.

The eligibility engine can inject one extra instruction into the prompt: when
a posting passed via sponsorship + relocation, the draft is told to highlight
relocation readiness (using only the facts the bank provides). This is a
prompt hint derived from a deterministic verdict, not another AI decision.

---

## 7. The eligibility engine

The newest stage, and the clearest expression of the project's principles.
Four rules decide whether a posting is worth *any* effort, evaluated in a
fixed precedence order — veto beats pass, and cheaper checks run first:

1. **Existing work rights** (veto) — "citizens only", "must be authorized to
   work in {country}". Checked first because it is an absolute disqualifier.
2. **Country-bounded remote** (veto) — "Remote — US only", "must reside in
   Poland" — with an exemption: if the posting is bounded to a country listed
   in `work_authorized_countries`, it passes via `authorized_country_remote`.
3. **Sponsorship + relocation** (pass) — on-site/hybrid roles offering visa
   sponsorship *and* relocation assistance; flags the draft to highlight
   relocation readiness.
4. **Worldwide remote** (pass) — global remote, hire-anywhere, EOR and
   contractor arrangements.

Anything with no signal is `unknown` and passes through — **a veto must be
evidence, not a guess** — unless the operator opts into rejecting unknowns.

Implementation details that matter:

- **Pure and offline.** `EvaluateEligibility(job, rules)` is a function of
  its inputs only: no network, no LLM, no clock. It runs in microseconds,
  which is what allows it to sit in front of the drafter for every posting.
- **Phrase matching with `{country}` templating.** Phrase lists are matched
  case-insensitively on word boundaries against the normalised
  title+location+description blob; `{country}` in a phrase expands to any
  recognised country name or unambiguous code. A negative-sentinel guard
  suppresses pass signals preceded by a negation within a small window ("no
  relocation support" must never count as relocation support) — veto phrases
  are matched as written, because "no visa sponsorship" *is* the signal.
- **Verdicts quote their evidence.** Every verdict carries the matched
  phrases and the countries found, stored on the job row. A wrong veto can be
  traced to the exact sentence that caused it, and the tester on `/settings`
  exists precisely to try a posting before trusting the rules.
- **Everything is configurable.** Rule switches, the work-authorized country
  list, every phrase list and the relocation hint live in
  `settings.yaml → eligibility`, editable in the UI. Defaults embody the
  stated policy, so the engine works before any configuration exists.
- **Hand-set states are respected.** `eligibility_applied` marks verdicts the
  engine set. If the operator moves a card by hand, the engine stops owning
  it; the re-check endpoint skips operator-decided cards. Automation must
  know what it owns.
- **Re-check without re-scraping.** `POST /api/eligibility/reapply` re-runs
  the rules over stored jobs, so a policy change takes effect immediately —
  restoring previously vetoed postings — without hitting any job board again.

---

## 8. Concurrency, failures, and honesty

- **Per-source isolation.** One scraper's panic or timeout costs one source's
  postings, never the run. Failures are logged with reasons and surfaced in
  the run summary and in `/api/sources/health`.
- **Rate budgeting, not scraping bravado.** Sources with published guidance
  (Remotive: ~4 requests/day) are budgeted accordingly; sites that restrict
  automated access (Bayt, SEEK) are opt-in and may stay blocked. The system
  says what it did and what it skipped.
- **Login-walled platforms are out by principle.** LinkedIn/Wellfound session
  scraping is fragile and ToS-hostile; the source list leans on public APIs
  and boards instead. Scope decisions are documented, not accidental
  omissions.
- **Human in the loop.** The app finds, scores, drafts and tracks. Submission
  is a human decision. This keeps the system honest about what it has done —
  there is no background process sending applications on your behalf.

---

## 9. Testing principles

- Scraper and service tests run on local fixtures; no test hits the network.
- Store tests run the real migrations and real SQL against a temporary
  database — the persistence layer is tested against SQLite's actual
  behaviour, not a mock of it.
- The API has an OpenAPI coverage gate: every registered route must appear in
  `openapi.yaml`, so the documentation cannot drift from the code.
- The eligibility engine is table-driven over the exact policy statements
  ("Remote — US only" vetoes, "work from anywhere" passes, exemption honours
  the country list), so the tests read like the specification.

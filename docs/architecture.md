# Job Agent Backend Architecture

This is the backend for the self-hosted job search pipeline: discovery, scoring, drafting, and tracking. It is written in Go and exposes a documented HTTP API that the Next.js dashboard and any future client consume.

Treat this document as the **current** description of the codebase. The older spec in `job-agent-spec.md` is planning context -- it explains the *why* behind several decisions, but it is not a live description of what is built.

---

## 1. Overview

The backend has four jobs:

1. **Discover** -- run scrapers against public job boards and curated company pages.
2. **Score** -- evaluate each posting against compensation, remote/relocation criteria, and the candidate's track, producing a match result and a recommended pipeline status.
3. **Draft** -- optionally generate cover letters and ATS answers from the editable project bank.
4. **Track** -- expose the pipeline over HTTP so the dashboard and bot can read and mutate state (move jobs between columns, review drafts, trigger applies).

Everything is local-first. There is no cloud dependency, no account, and no telemetry. The only external calls are the job sources you enable and, optionally, an LLM endpoint for drafting and resume analysis.

---

## 2. Repository layout

```
job-bot/
|-- backend/
|   |-- cmd/server/main.go      # entrypoint: load config, open DB, build router, serve
|   |-- internal/
|   |   |-- api/                # HTTP handlers + OpenAPI spec
|   |   |-- db/                 # SQLite, migrations, queries
|   |   |-- models/             # shared structs (Job, Match, settings shapes)
|   |   |-- scrapers/           # one package per job source + registry
|   |   |-- services/           # ingest pipeline, settings, resume analysis, drafting
|   |   `---- textutil/           # small helpers: HTML->text, trimming
|   |-- go.mod / go.sum
|   |-- Dockerfile
|   `---- .env.example
|-- frontend/                   # Next.js 15 dashboard
|-- docs/
|   |-- architecture.md         # how the code is organised
|   |-- tech-stack.md           # stack choices, DB rationale, algorithms & principles
|   `-- job-agent-spec.md       # planning context (not a live description)
|-- data/                       # runtime data: DB, editable YAML -- gitignored
|-- resumes/                    # candidate PDFs -- gitignored
`---- README.md
```

A few conventions worth calling out:

- **`cmd/server/`** is the only `main` package. It stays thin: load config, open the database, build the router, listen. No business logic lives here.
- **`internal/`** is compiler-enforced privacy. Packages under it can only be imported by code in this module, so the store, handlers, and services are private by default.
- **`data/` and `resumes/` live at the repo root**, not under `backend/`, so a Docker bind mount and a local run share the same files without path gymnastics.
- **Editable YAML** (`data/settings.yaml`, `data/target_companies.yaml`, `data/project_bank.yaml`) is the user-owned configuration surface. The app writes settings atomically; the rest is hand-edited.

---

## 3. API and docs

The API is a flat set of `GET/POST/PATCH` handlers under `internal/api`. There is no framework magic -- just `net/http` + a small router. OpenAPI is **hand-written** in `backend/internal/api/openapi.yaml` and served from `/openapi.yaml`. The dashboard and any client can load Swagger UI from `/docs` while the server runs.

This is deliberately low-tech. Go's `net/http` does not auto-generate specs the way FastAPI does, and for a project this size a small YAML file next to the handlers is clearer than a comment-driven generator. If the API surface grows a lot, that tradeoff can be revisited.

Endpoints that matter day-to-day:

- `/api/jobs` -- list and search postings, with filters for track, status, score.
- `/api/jobs/:id` -- read one posting; `PATCH` updates its status (drag-and-drop on the board calls this).
- `/api/settings` -- read and write source toggles and target regions.
- `/api/sources` -- metadata for every known source: label, kind, supported countries, whether it needs an API key, and whether it is opt-in.
- `/api/sources/health` -- probe each source and report what actually came back, so dead company pages and blocked sites become visible.
- `/api/stats` -- pipeline counts.
- `/api/resumes` -- resume directory and per-track resolution.
- `/api/resumes/analyze` -- trigger AI resume analysis for a file or the whole directory.

---

## 4. Data model

The core tables are small:

- **`jobs`** -- one row per discovered posting: source, source id, url, title, company, location, remote flag, salary range, description, posted-at, and current pipeline status.
- **`matches`** -- one row per scored job: score, assigned track, matched and missing skills, and the human-readable reasons for the score.
- **`resume_profiles`** -- one row per resume file: content hash, AI-extracted track and keywords when available, otherwise the filename-based fallback. Used by the matcher to prefer the right track and to add AI keyword signal.

Status flow is explicit and finite: `discovered`, `scored`, `queued`, `approved`, `applied`, `interview`, `rejected`. The board columns map to subsets of these, and drag-and-drop on the dashboard is just a status update.

---

## 5. Sources and settings

Every job source is described by a **spec** in `internal/scrapers/registry.go`: a stable name, a human label, a kind (`board`, `aggregator`, `feed`, or `gated`), the countries it can target, any API keys it needs, an opt-in flag when the site restricts automated access, and a note about paywalls or attribution.

Adding a source is one registry entry plus one scraper file implementing a small interface. The ingest loop reads the registry, applies the user's settings and target regions, and runs only what is enabled. A source with no supported country in the active region list is skipped with a logged reason, not silently ignored.

Settings live in `data/settings.yaml` and have two parts:

- **Source toggles** -- which scrapers run.
- **Target regions** -- lowercase ISO-3166 alpha-2 country codes. These scope country-aware sources and the recruiter directory. The default list is `ie`, `gb`, `pk`, `ae`, `sa`, `au`, `nz`, plus `us` and `ca`.

Source-specific notes are honored, not hidden:

- **RemoteOK and the Crypto/Web3 feed** are flagged as paywalled because applying can require RemoteOK Premium or a login.
- **Jobicy** requires crediting with a link to `jobicy.com`.
- **Himalayas** and **Arbeitnow** are free and have no special obligation.
- **Remotive** is free in principle but its own guidance is to stay around 4 requests/day, so it is rate-budgeted, not run on every ingest.
- **Bayt**, **SEEK AU**, and **SEEK NZ** are **gated opt-ins**: the sites restrict automated access, so they stay off unless you explicitly enable them with an environment flag, and even then they may be blocked. SEEK in particular only permits `?keywords` search URLs under its robots.txt -- the internal `/api/jobsearch` and `/graphql` endpoints are explicitly disallowed, so we do not call them.

---

## 6. Scoring and resume analysis

Matching is built in two layers:

1. **Heuristics** -- compensation, remote/relocation, track signals from title and description, and a small set of track-specific keywords. This is the baseline and works without any LLM.
2. **AI supplement** -- when `OPENAI_API_KEY` is set, resumes are analyzed once per file (cached by content hash) to extract track, skills, keywords, seniority, and a one-line summary. The matcher then adds a bounded keyword-signal boost when those keywords appear in a posting.

The AI layer is a **supplement**, not a replacement. Without a key, the matcher is exactly as good as the heuristics. With a key, it gets better at preferring the right track and at surfacing relevant skills, but the score reasons always say which signal contributed, so a blank score is informative rather than mysterious.

---

## 7. Drafting

Drafting is optional and key-gated. When enabled, the drafter uses the editable project bank (`data/project_bank.yaml`) to ground cover letters and ATS answers in real, editable bullets instead of inventing experience. The bank is the canonical source of truth for what gets written about the candidate -- not the LLM, not a hidden profile.

---

## 8. How to explore it

From the repo root:

```bash
cd backend
go run ./cmd/server
```

Useful URLs while it runs:

- `http://localhost:8000/docs` -- interactive API docs
- `http://localhost:8000/openapi.yaml` -- raw spec
- `http://localhost:8000/health` -- liveness
- `http://localhost:8000/api/stats` -- pipeline counts

From another terminal:

```bash
cd backend
go test ./internal/...
```

The scraper and service tests use local fixtures and never hit the network. The API tests that touch settings and sources read `data/settings.yaml` from the repo root, so they work from a clean checkout.

---

## 9. What is phased out, and what isn't

A few ideas from the original spec did not make it into the code, and that is intentional:

- **LinkedIn and Wellfound** are not scraped. Session-based scraping of login-walled platforms is fragile, ToS-hostile, and tends to break quickly. The source list instead leans on public boards, Greenhouse/Lever/Ashby company pages, and aggregators that expose real APIs.
- **Autonomous applying** is not built out. The app tracks and drafts; actual submission is a human decision in the dashboard. This keeps the system honest about what it has done.
- **Telegram bot** is not present in this backend. Notification and command surface is a future extension; the API already exposes everything the bot would need.

These are not missing pieces waiting to be filled in -- they are deliberate choices about scope and honesty. The app finds, scores, drafts, and tracks. What you do with a scored, drafted job is up to you.

---

If something in the tree feels arbitrary, it is usually one of three things: standard Go layout (`cmd`/`internal`), a deliberate Phase 1 simplicity choice (a single `Store` instead of a full service layer), or a scope decision documented above. All three are intentional.

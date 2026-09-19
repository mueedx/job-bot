# Job Agent

[![CI](https://github.com/mueedx/job-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/mueedx/job-bot/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-15-black.svg)](frontend/package.json)

An autonomous job-finding pipeline you run yourself. It scrapes public job
boards, scores every posting against your skills, routes it to the right resume,
optionally writes a grounded cover letter, and shows everything in a kanban
dashboard for you to review.

**It does not apply for you.** Submitting applications is deliberately not
implemented yet — the API returns `501 Not Implemented` and marks the job
`approved` so you can finish it by hand. Nothing is ever sent on your behalf.

- **Find** — public Greenhouse, Lever, Ashby, RemoteOK and CryptoJobs boards
- **Match** — keyword scoring plus remote/compensation heuristics, routed to one
  of your resume tracks (`fullstack`, `blockchain`, `fde`)
- **Draft** — optional OpenAI-compatible cover letters written from *your* facts
  file, with instructions not to invent anything
- **Review** — Next.js control center: pipeline board, analytics, per-job editor

## How it works

```
 public job boards                your resumes/              optional
 +-------------------+          +----------------+      +--------------+
 | Greenhouse        |          | fullstack.pdf  |      | OpenAI-      |
 | Lever / Ashby     |  scrape  | blockchain.pdf |      | compatible   |
 | RemoteOK          | -------> | fde.pdf        | ---> | cover letter |
 | CryptoJobs        |          | default.pdf    |      | drafting     |
 +-------------------+          +----------------+      +------+-------+
            |                            ^                     |
            v                            | route by track      v
      score + filter --------------------+              kanban dashboard
      (match, thresholds, age)                          (you review + apply)
```

1. **Ingest** pulls postings from public board APIs and de-duplicates by URL.
2. **Match** scores each job (0–1) from keyword hits, remote/relocation and
   salary heuristics, then picks a resume track.
3. **Draft** (optional) writes a cover letter into the application record.
4. **Review** happens in the dashboard: approve, edit, switch track, or discard.

## Requirements

- **Docker path (recommended):** Docker with the Compose plugin.
- **Local path:** Go 1.25+ and Node.js 22+.
- At least one resume PDF in `resumes/` — see [Add your resumes](#add-your-resumes).
- Optional: an OpenAI-compatible API key for cover letters, and a Telegram bot
  for alerts. Everything else works without either.

## Quick start (Docker)

```bash
git clone https://github.com/mueedx/job-bot.git
cd job-bot

# 1. Your resumes: drop 1-3 PDFs into resumes/ (naming is flexible)
cp ~/Documents/My_Resume.pdf resumes/My_Fullstack_Resume.pdf

# 2. Optional: secrets for drafting/Telegram. Safe to skip.
cp .env.example .env

# 3. Build and run
docker compose up --build
```

Then open **http://localhost:3000** and press **Run search**. The first run takes
a minute or two while it walks every company board.

| URL | What it is |
|---|---|
| http://localhost:3000 | Dashboard (pipeline board) |
| http://localhost:3000/stats | Analytics |
| http://localhost:8000/docs | Swagger UI for the API |
| http://localhost:8000/health | Health check |

Your data lives in the repo's `data/` directory (bind-mounted to `/data`), so it
survives restarts:

```bash
docker compose down          # stop
docker compose up            # start again, data intact
docker compose logs -f api   # watch the API logs
```

No `.env` file is required — Compose starts fine without one.

> On Linux, files the container creates in `data/` are owned by `root`. If that
> gets in your way: `sudo chown -R "$USER" data`.

## Quick start (without Docker)

Two terminals:

```bash
# Terminal 1 — API (listens on :8000)
cd backend
go run ./cmd/server

# Terminal 2 — dashboard (listens on :3000)
cd frontend
cp .env.local.example .env.local
npm install
npm run dev
```

The server finds `data/` and `resumes/` automatically, whether you launch it from
the repo root or from `backend/`.

Run a search without opening the UI:

```bash
curl -X POST http://localhost:8000/api/ingest/run     # start
curl http://localhost:8000/api/ingest/status          # progress + logs
```

Add demo jobs to explore the dashboard before scraping anything:

```bash
curl -X POST http://localhost:8000/api/dev/seed
```

## Add your resumes

Put your PDFs in the **`resumes/`** directory at the repo root. That is the only
place the app reads from by default.

The matcher routes each job to one of three tracks — `fullstack`, `blockchain`,
`fde` — and each track resolves to a file in this order:

1. **Exact name wins:** `fullstack.pdf`, `blockchain.pdf`, `fde.pdf`.
2. **Name tokens:** any file whose name contains the track's keywords, so
   `Jane_Doe_Fullstack.pdf` or `Jane_Doe_Forward_Deployed.pdf` just work.
3. **`default.pdf`:** used for any track that has no file of its own — handy when
   you have a single general-purpose resume.
4. **A lone PDF:** if `resumes/` holds exactly one PDF that does not clearly
   belong to another track, it is used for every track.

| Track | Recognised name tokens |
|---|---|
| `fullstack` | `fullstack`, `web`, `react`, `nestjs`, `nodejs`, `typescript`, `node js`, `full stack` |
| `blockchain` | `blockchain`, `web3`, `solidity`, `evm`, `crypto`, `defi`, `substrate`, `smart contract` |
| `fde` | `fde`, `ai`, `agentic`, `mcp`, `rag`, `forward deployed`, `ai engineer`, `solutions architect`, `customer engineer` |

Words must match whole tokens, so `web3.pdf` is never mistaken for `web`.

Check what the server actually sees at any time:

```bash
curl http://localhost:8000/api/resumes
```

```json
{
  "dir": "resumes",
  "files": ["Jane_Doe_Blockchain.pdf", "Jane_Doe_Forward_Deployed.pdf", "Jane_Doe_Fullstack.pdf"],
  "tracks": {
    "blockchain": "resumes/Jane_Doe_Blockchain.pdf",
    "fde": "resumes/Jane_Doe_Forward_Deployed.pdf",
    "fullstack": "resumes/Jane_Doe_Fullstack.pdf"
  },
  "missing": [],
  "entries": [ "..." ]
}
```

If a track has no matching PDF, the stored path is a placeholder such as
`resumes/fde.pdf` and the track is listed under `missing` — the job still shows
up in the pipeline, with a resume you can add later.

- Keep PDFs elsewhere? Set `RESUME_DIR=/absolute/path/to/resumes`.
- **Your PDFs are gitignored.** `*.pdf` never leaves your machine.

## Make it yours

`data/` holds the two files worth editing (see [data/README.md](data/README.md)):

**`data/target_companies.yaml`** — which company boards to scrape. Seeded on first
run from `target_companies.yaml.example`; add or remove Greenhouse/Lever/Ashby
slugs and delete these if you want only the keyword-driven boards:

```yaml
greenhouse:
  - langchain
  - replicate
lever:
  - openai
ashby:
  - cursor
```

**`data/project_bank.yaml`** — the facts used to write cover letters. This file is
personal and is **not** seeded for you, because the drafter refuses to invent
anything:

```bash
cp data/project_bank.yaml.example data/project_bank.yaml
$EDITOR data/project_bank.yaml
```

Until it exists, drafting returns a clear error instead of a fabricated letter.
Drafting also needs `OPENAI_API_KEY`, and is entirely optional — matching,
routing, and the dashboard work without it.

Prefer a clean slate? Delete `data/jobs.db` and restart, or
`docker compose down -v && rm -f data/jobs.db`.

## Configuration

Copy `.env.example` to `.env`; every value has a working default. Missing `.env`
is fine — Compose and the server both start without it.

| Variable | Default | Purpose |
|---|---|---|
| `ADDR` | `:8000` | API listen address |
| `DATA_DIR` | auto (`./data`) | SQLite + editable YAML |
| `RESUME_DIR` | auto (`./resumes`) | Where resume PDFs live |
| `DASHBOARD_URL` | `http://localhost:3000` | Used for CORS + Telegram links |
| `CORS_ALLOWED_ORIGINS` | falls back to `DASHBOARD_URL` | Comma-separated extra browser origins |
| `INGEST_CRON_ENABLED` | `false` | Run a search on a timer |
| `INGEST_CRON_INTERVAL_HOURS` | `24` | Timer interval |
| `JOB_MAX_AGE_DAYS` | `0` (no limit) | Drop postings older than this; unknown dates always kept |
| `AUTO_APPLY_MIN_SCORE` | `0.80` | Score at or above which a job is marked `queued` (nothing is auto-submitted) |
| `DRAFT_MIN_SCORE` | `0.75` | Score at or above which a search spends an LLM call drafting |
| `OPENAI_API_KEY` | _(empty)_ | Enables drafting. Without it, `/draft` returns `503` |
| `OPENAI_API_BASE_URL` | `https://api.openai.com/v1` | Any OpenAI-compatible endpoint (OpenRouter, Groq, Together, Ollama, vLLM…); a full `…/chat/completions` URL works too |
| `LLM_MODEL` | `gpt-4o-mini` | Model name passed through to that endpoint |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | _(empty)_ | Enables high-match alerts |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8000` | API URL the **browser** calls; baked in at build time |
| `DAILY_APPLICATION_LIMIT` | `8` | Reserved for the Phase 5 submitter |
| `SUBMISSION_JITTER_MIN_SECONDS` / `_MAX_SECONDS` | `45` / `90` | Reserved for the Phase 5 submitter |
| `EXECUTION_STAGE` | `calibration` | Reserved; `autonomous` is not implemented |
| `LINKEDIN_LI_AT`, `WELLFOUND_SESSION` | _(empty)_ | Deferred, unused — see [Roadmap](#roadmap--status) |

`NEXT_PUBLIC_API_URL` is compiled into the frontend, so changing it requires a
rebuild (`docker compose up --build web`, or restart `npm run dev`).

## Using the dashboard

| Page | What you do there |
|---|---|
| `/` — **Pipeline** | Run searches, watch live progress and logs, and work the board: `discovered` → `scored` → `queued` → `applied` → `interview` / `rejected`. **Drag cards between columns** to update a job's status (optimistic — a failed save snaps the card back with an error) |
| `/jobs/{id}` | Read the match score, matched/missing skills, and score reasons; read and edit the cover letter; **Switch Track** to re-route the resume; **Approve**; **Discard** |
| `/stats` — **Analytics** | Totals by status (applications, interviews, rejections) |

**About “Approve & Submit”:** the button exists, but submission is not
implemented. It marks the job `approved`, returns `501`, and logs the honest
message *“application was NOT sent”*. Finish that application yourself.

### Job sources & paywalls

Each job shows where it came from. Some boards gate applying behind a
subscription — jobs from those sources carry an amber **premium** badge on the
board and the job page, so you know before you click through:

| Source | Applying | Badge |
|---|---|---|
| Greenhouse / Lever / Ashby | Company's own ATS form, free | — |
| RemoteOK | May require **RemoteOK Premium** or a login | amber `premium` |
| CryptoJobs | Mirrors RemoteOK listings; same premium gate | amber `premium` |

## API

Interactive docs: **http://localhost:8000/docs** (spec at
http://localhost:8000/openapi.yaml).

| Method | Path | What it does |
|---|---|---|
| `GET` | `/health` | Liveness |
| `GET` | `/api/jobs` | List jobs; `?status=` filter, `?limit=` (max 500), `?enrich=1` adds score/track |
| `POST` | `/api/jobs` | Create a job manually |
| `GET` | `/api/jobs/{id}` | One job |
| `PATCH` | `/api/jobs/{id}` | Update status/location/salary/etc. |
| `GET` | `/api/jobs/{id}/detail` | Job + match + application |
| `PUT` | `/api/jobs/{id}/application` | Save cover letter, or `{"track":"fde"}` to switch resume |
| `POST` | `/api/jobs/{id}/draft` | Generate a cover letter (needs `OPENAI_API_KEY`) |
| `POST` | `/api/jobs/{id}/approve` | Mark `approved` — returns `501`, does not submit |
| `POST` | `/api/jobs/{id}/discard` | Mark `rejected` |
| `POST` | `/api/ingest/run` | Start a search; `202` accepted, `409` if one is running |
| `GET` | `/api/ingest/status` | Phase, progress, counters, and log lines |
| `GET` | `/api/resumes` | Which PDF each track resolves to, and what is missing |
| `GET` | `/api/stats` | Counts by status |
| `POST` | `/api/dev/seed` | Insert demo jobs for exploring the UI |
| `POST` | `/api/apply/{id}` | Submission stub — always `501` |

```bash
# A complete run from the command line
curl -X POST http://localhost:8000/api/ingest/run
curl http://localhost:8000/api/ingest/status | jq '.phase, .inserted, .scored, .drafted'
curl 'http://localhost:8000/api/jobs?enrich=1&limit=5' | jq '.[0] | {title, company, score, track}'
```

## Project layout

```
job-bot/
├── resumes/              # YOUR PDFs — gitignored, mounted read-only in Docker
├── data/                 # YOUR SQLite DB + editable YAML — gitignored
│   ├── target_companies.yaml.example
│   └── project_bank.yaml.example
├── backend/              # Go 1.25 API
│   ├── cmd/server/       # entrypoint: config → DB → router → listen
│   └── internal/
│       ├── api/          # chi handlers + embedded OpenAPI spec
│       ├── db/           # SQLite open/migrate/queries
│       ├── models/       # shared structs
│       ├── resumes/      # resume file discovery + track resolution
│       ├── scrapers/     # Greenhouse, Lever, Ashby, RemoteOK, CryptoJobs
│       └── services/     # matching, ingestion, drafting, Telegram bot
├── frontend/             # Next.js 15 + Tailwind dashboard
├── docs/                 # architecture notes and the original spec
├── specs/                # Spec Kit feature specs
├── docker-compose.yml
└── .env.example
```

New to Go, or coming from NestJS/FastAPI? [docs/architecture.md](docs/architecture.md)
explains the layout and the patterns used.

## Testing

```bash
cd backend  && go build ./... && go vet ./... && go test ./...
cd frontend && npm run lint && npm run build
```

## Troubleshooting

**A search finishes but the board is empty.**
Check the log lines in the search panel. Boards move and rate-limit; a source can
fail while others succeed (`source_errors` in `/api/ingest/status`). Also check
`JOB_MAX_AGE_DAYS` — the default `0` keeps everything, but if you set it to `7`,
older postings are dropped. Make sure `data/target_companies.yaml` exists.

**“OPENAI_API_KEY not set” when drafting.** Drafting is optional. Set the key (and
optionally `OPENAI_API_BASE_URL`) in `.env` and restart to enable it.

**“cannot read …/project_bank.yaml”.** The drafter will not invent your history.
`cp data/project_bank.yaml.example data/project_bank.yaml` and fill it in.

**The dashboard shows “Failed to fetch” / nothing loads.** `NEXT_PUBLIC_API_URL`
is baked into the frontend at build time. If the API is not on
`http://localhost:8000`, set it and rebuild: `docker compose up --build web`.

**A job shows `resumes/fde.pdf` as its resume.** No PDF matched that track. Run
`curl http://localhost:8000/api/resumes` to see what was found and which tracks
are `missing`, then add a file — see [Add your resumes](#add-your-resumes).

**“port is already allocated”.** Something else owns 3000 or 8000. Change the
published ports in `docker-compose.yml`, or `ADDR` for the API.

**Start over.** `docker compose down && rm -f data/jobs.db`, then start again.

**Root-owned files in `data/` (Linux).** `sudo chown -R "$USER" data`.

## Roadmap & status

| Phase | Status |
|---|---|
| 1 — Go API, SQLite, Swagger | ✅ |
| 2 — Public board ingestion + keyword matcher | ✅ |
| 3 — OpenAI cover letters + Telegram alerts | ✅ (optional) |
| 4 — Next.js control center | ✅ |
| 5 — Application submitter | ❌ Not implemented (`501`) |

Deliberately out of scope for now: **LinkedIn and Wellfound scraping.** Both
require a logged-in session, and automating them is likely to breach their terms
of service and get accounts restricted. The env placeholders exist but are
unused.

Please scrape responsibly: this project only talks to public, documented board
APIs, and you are responsible for how you use the data.

## Contributing

Issues and pull requests are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
For security reports, use [SECURITY.md](SECURITY.md) rather than a public issue.
This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE).

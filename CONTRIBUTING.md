# Contributing

Thanks for taking a look. This is a small, pragmatic project — bug reports and
focused pull requests are very welcome.

## Getting set up

```bash
git clone https://github.com/mueedx/job-bot.git
cd job-bot
make setup          # creates .env and data/target_companies.yaml
# add at least one PDF to resumes/
make docker         # or: make dev-api + make dev-web in two terminals
```

Requirements: Docker with Compose, or Go 1.25+ and Node.js 22+ for local work.

## Before you open a pull request

```bash
make test           # go build/vet/test + npm lint/build
```

Both must pass. CI runs the same commands.

## What makes a good change here

- **Keep it honest.** The project never claims to submit applications it cannot
  submit. Don't add fake progress states or silently swallow errors — the
  `501 Not Implemented` response and the "application was NOT sent" message are
  deliberate.
- **Don't invent candidate facts.** Anything the drafter says must come from
  `data/project_bank.yaml`.
- **Keep personal data out.** `resumes/*.pdf`, `data/*.yaml`, `.env`, `.kilo/`,
  `.cursor/` and `.specify/` are gitignored, and CI fails if they are ever
  committed.
- **Respect the boards.** New scrapers must use public, documented APIs. Do not
  add LinkedIn or Wellfound session scraping — it likely breaches their terms of
  service.
- **Match the existing style.** Small functions, comments that explain *why*,
  table-driven tests next to the code they cover.

## Working on the frontend

```bash
cd frontend
npm run lint        # ESLint
npm run typecheck   # tsc --noEmit
npm run build       # production build (also type-checks)
```

The pipeline board moves jobs with native HTML5 drag & drop; the status change
goes through `api.updateJobStatus` (`PATCH /api/jobs/{id}`) with an optimistic
update that reverts on failure.

When you add a job source, check its apply flow and, if it gates applying
behind a subscription or login, add it to `PAYWALLED_SOURCES` in
`frontend/src/lib/api.ts` so the dashboard can badge it. Leave sources you
have not verified off the list — a missing badge is honest, a wrong one is not.

## Adding a scraper

1. Implement `scrapers.Scraper` (`Name()` and `Fetch(ctx)`) in
   `backend/internal/scrapers/`.
2. Normalise into `RawJob`, including `PublishedAt` when the source exposes it
   (`parseISOTime` / `parseEpochTime` helpers exist for this).
3. Register it in the list in `backend/internal/services/ingest.go`.
4. Add a test with a captured JSON fixture — no live network calls in tests.

## Reporting bugs

Include what you ran, what you expected, and what happened. For ingestion issues,
paste the log lines from the search panel or `GET /api/ingest/status`; board APIs
change often and we cannot fix what we cannot see.

Please don't report security problems in public issues — see
[SECURITY.md](SECURITY.md).

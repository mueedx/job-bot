# API Contracts: Control Center

Base: `http://localhost:8000`

## Existing (keep)

- `GET /health`
- `GET /api/jobs` — list; add `?enrich=1` for score/track
- `GET /api/jobs/{id}`
- `PATCH /api/jobs/{id}`
- `POST /api/jobs`
- `POST /api/apply/{id}` — may 501
- `GET /api/stats`
- `GET /docs`, `GET /openapi.yaml`

## New

### `GET /api/jobs/{id}/detail`

Returns `{ job, match, application }` (match/application null if missing).

### `PUT /api/jobs/{id}/application`

Body: `{ cover_letter?: string, track?: "fullstack"|"blockchain"|"fde" }`  
Upserts application; sets `resume_path` from track; default submission_method `manual`.

### `POST /api/jobs/{id}/discard`

Sets status `rejected`. Returns updated job.

### `POST /api/jobs/{id}/approve`

Sets status `approved`. Attempts submission; if unavailable → **409 or 501** with `{ detail }` explaining not ready. Does not set `applied` on failure.

### `POST /api/dev/seed`

Idempotent demo jobs/matches/applications for local UI.

### `GET /api/stats` (extend)

Include `daily_limit` integer from env for analytics guidance.

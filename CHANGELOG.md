# Changelog

Notable changes to this project. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Pipeline board is now interactive: drag job cards between columns to change
  their status (optimistic UI with revert + error banner on failure), via the
  existing `PATCH /api/jobs/{id}`.
- Paywall indicator on source links: jobs from RemoteOK and CryptoJobs carry an
  amber "premium" badge on the board and the job page, warning that applying
  may require a RemoteOK Premium subscription or login. Free ATS sources
  (Greenhouse, Lever, Ashby) are unbadged.
- `engines` field in `frontend/package.json` (Node ≥ 22) and a Dependabot
  config for Go, npm, Docker and GitHub Actions updates.

### Changed

- The cover-letter prompt addresses the candidate by the name in
  `data/project_bank.yaml` instead of a hardcoded name, falling back to a
  neutral phrasing when it is missing.
- Docs and tests use neutral placeholder names (`Jane_Doe_*.pdf`) instead of
  personal ones.

### Fixed

- The CI "documented endpoints" check compares README paths against routes as
  registered inside the `/api` route group. As written it always failed, which
  would have blocked the first push.

## Previous work (this cycle)

### Added

- `resumes/` at the repository root, with automatic track resolution: exact
  `<track>.pdf` names win, then filename tokens (`Jane_Doe_Fullstack.pdf`), then
  `default.pdf`, then a lone PDF. `RESUME_DIR` overrides the location.
- `GET /api/resumes` — which PDF each track resolves to, and which are missing.
- Startup log lines reporting the resolved resume directory and missing tracks.
- `data/` at the repository root holding the SQLite database and editable YAML,
  with `*.example` templates for `project_bank.yaml` and `target_companies.yaml`.
- `CORS_ALLOWED_ORIGINS` (falls back to `DASHBOARD_URL`).
- `JOB_MAX_AGE_DAYS` to limit how old a posting can be, and `posted_at` on jobs.
- ESLint configuration and a `typecheck` script for the dashboard.
- Docker: bind-mounted `data/` and `resumes/`, an API health check, and a
  `web` service that waits for a healthy API.
- Project docs: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`,
  `docs/architecture.md`, `data/README.md`, `resumes/README.md`, a `Makefile`,
  and GitHub Actions CI.
- Resume resolution tests, ingest tests, and a migration test.

### Changed

- Docker build context moved to the repository root; the image no longer bakes in
  resume PDFs or personal config.
- The dashboard derives a job's resume track from the stored filename using the
  same token rules as the backend.
- The dashboard's API docs link is derived from `NEXT_PUBLIC_API_URL` instead of
  a hardcoded localhost URL.
- `DATA_DIR` and `RESUME_DIR` are auto-detected, so the server behaves the same
  when started from the repository root or from `backend/`.
- `npm run lint` uses the ESLint CLI; `next lint` is deprecated in Next.js 16.

### Fixed

- Local runs from `backend/` and Docker runs now read the same `data/` and
  `resumes/` directories.
- `OPENAI_API_BASE_URL` was missing its `=` in `.env.example`.
- The scraper `User-Agent` no longer points at `localhost`.
- Drafting reports an actionable error when `project_bank.yaml` is absent instead
  of returning a bare file-not-found.

### Removed

- Personal resume PDFs and Windows `*:Zone.Identifier` artefacts.
- `backend/resumes/` and `backend/data/` (moved to the repository root).

[Unreleased]: https://github.com/mueedx/job-bot/commits/main
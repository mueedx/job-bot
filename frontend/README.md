# Dashboard (frontend)

Next.js 15 (App Router) control center for the job pipeline: run searches,
review matches, edit drafted cover letters, and track applications.

```bash
cp .env.local.example .env.local   # points NEXT_PUBLIC_API_URL at the API
npm install
npm run dev                        # http://localhost:3000
```

The API must be reachable at `NEXT_PUBLIC_API_URL` (default
`http://localhost:8000`).

## Scripts

| Script | What it does |
|---|---|
| `npm run dev` | Dev server with hot reload |
| `npm run build` | Production build (also type-checks) |
| `npm run lint` | ESLint (`next/core-web-vitals` + TypeScript rules) |
| `npm run typecheck` | `tsc --noEmit` |

## Where things live

- `src/app/` — routes (`/` pipeline, `/jobs/[id]` detail, `/stats` analytics)
- `src/components/` — `AppNav`, `JobCard`, `KanbanColumn`, `IngestPanel`,
  `ActionBar`, `CoverLetterEditor`
- `src/lib/api.ts` — the only place that talks to the API, plus
  - `paywallNotice(source)` — paywall warnings per job source (see the
    `PAYWALLED_SOURCES` map; verified free ATS sources are deliberately absent)
  - `columnForStatus` / `statusForColumn` — the board-column ⇄ status mapping
- Drag & drop is native HTML5: `JobCard` sets `text/job-id` on `dragstart`,
  `KanbanColumn` reads it on `drop` and calls `onMoveJob`, which does an
  optimistic move then `PATCH /api/jobs/{id}` (reverts on failure). No DnD
  library.

Layout and theme: dark, keyboard-friendly, IBM Plex Sans/Mono via `@fontsource`
(no external font CDN).

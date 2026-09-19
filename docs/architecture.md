# Go Learning Guide — Job Agent Backend

This guide explains **why** this repo is laid out the way it is, and **which Go patterns** it uses. It is written for someone who knows NestJS/Next.js and is learning Go.

FastAPI gave you interactive docs at `/docs` for free. In Go there is no built-in equivalent, so we wire **OpenAPI + Swagger UI** ourselves (see below).

---

## 1. FastAPI `/docs` → Go equivalent

| FastAPI | This Go project |
|---|---|
| Auto OpenAPI from Pydantic/routes | Hand-written [`backend/internal/api/openapi.yaml`](../backend/internal/api/openapi.yaml) |
| `/docs` Swagger UI | `/docs` — HTML page loading Swagger UI from CDN |
| `/openapi.json` | `/openapi.yaml` — embedded spec file |
| `/redoc` | Not shipped (you can add later) |

**Why hand-written OpenAPI?** Go’s `net/http` does not introspect handlers the way FastAPI does. Tools like [swaggo](https://github.com/swaggo/swag) can generate a spec from code comments; for Phase 1 a small YAML next to the handlers is clearer and enough.

Open in the browser while the server runs: [http://localhost:8000/docs](http://localhost:8000/docs). Visiting `/` redirects there.

---

## 2. High-level layout (`cmd` + `internal`)

```
backend/
├── cmd/server/main.go     # process entrypoint only
├── internal/
│   ├── api/               # HTTP layer (chi handlers + OpenAPI)
│   ├── db/                # SQLite open, migrate, queries
│   ├── models/            # plain data structs (shared shapes)
│   ├── resumes/           # track → PDF resolution (RESUME_DIR, auto-detect)
│   ├── scrapers/          # one file per public job board
│   └── services/          # ingest pipeline + cover-letter drafter
├── go.mod                 # module identity + dependencies
├── Dockerfile             # backend image
└── .env.example
data/                      # runtime data (DB, editable YAML) — root level
resumes/                   # your PDFs — root level, gitignored
frontend/                  # Next.js 15 dashboard
```

> The runtime data and resumes live at the **repository root** (not under
> `backend/`) so a Docker bind mount and a local run share the same files.

### How a resume is picked

`internal/resumes` resolves a track to a PDF at request time — nothing is
hardcoded. `resumes.Path(track)` tries, in order: an exact `<track>.pdf`, a
filename whose tokens suggest the track (`Jane_Doe_Fullstack.pdf`), `default.pdf`,
then a lone PDF. The directory comes from `RESUME_DIR`, else `./resumes`, else
`../resumes`. `GET /api/resumes` reports the resolution, and startup logs any
missing tracks. See [resumes/README.md](../resumes/README.md).

### Why `cmd/server`?

In Go, **`package main` + `func main()`** is how you get a runnable binary. Putting it under `cmd/<name>/` is the community convention (used by Kubernetes, Hugo, etc.):

- One folder per binary (`cmd/server`, later maybe `cmd/scraper`).
- `main` stays thin: load config → open DB → build router → listen.
- Business logic does **not** live in `main`.

Think of `cmd/server` as Nest’s `main.ts` — bootstrapping only.

### Why `internal/`?

Go’s compiler enforces a rule: **code under `internal/` can only be imported by packages inside the same module** (roughly “this repo”). That keeps your DB store and handlers private so random other projects cannot depend on them.

Contrast NestJS, where everything is importable unless you discipline yourself. Here the filesystem + compiler do that for you.

### Why not a flat `app/` like FastAPI?

Python projects often use `app/main.py` + `app/api/`. That works. Go style prefers:

- `cmd` = binaries  
- `internal` = private library code  
- optional `pkg` = public libraries (we don’t need it yet)

---

## 3. Patterns used in this codebase

### Pattern A — Thin handlers, fat store (layered architecture)

```
HTTP request → api handler → db.Store → SQLite
                ↑ JSON/errors     ↑ SQL only
```

- **`internal/api`**: parse request, validate basics, call store, write JSON status codes.
- **`internal/db`**: all SQL. Handlers never write raw SQL.
- **`internal/models`**: structs with `json` and `db` tags — shared DTOs/entities.

This is close to Nest’s **Controller → Service → Repository**, except we currently merge “service + repository” into `Store` because Phase 1 has no domain logic yet. When matcher/drafter arrive, they go in `internal/services` and call `Store`.

### Pattern B — Dependency injection via struct fields (not a DI framework)

```go
type Server struct {
    Store *db.Store
}
```

`main` constructs `Store`, puts it on `Server`, passes `Server` methods as handlers. No Nest-style `@Injectable()` container — Go usually does **constructor injection by hand**. That stays readable until the graph gets huge.

### Pattern C — Chi router (composable `net/http`)

We use [chi](https://github.com/go-chi/chi), not Gin/Echo, because it sits on Go’s standard `http.Handler` interface:

- Middleware is just `func(http.Handler) http.Handler`.
- Easy to test with `httptest`.
- Nested routes (`/api/...`) mirror how you’d group Nest controllers.

FastAPI ≈ decorators on functions. Chi ≈ explicit route table in `router.go`.

### Pattern D — Repository / Store over an SQL library

[`sqlx`](https://github.com/jmoiron/sqlx) extends `database/sql` with struct scanning (`db` tags). We did **not** use a full ORM (GORM) so the SQL stays obvious and migrations stay a plain `schema.sql`.

SQLite driver: [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) — **pure Go**, no CGO. That matters on WSL and CI where C toolchain friction is annoying.

### Pattern E — `embed` for static SQL / OpenAPI

```go
//go:embed schema.sql
var schemaSQL string
```

and similarly for `openapi.yaml`. The file is compiled into the binary. No “forgot to copy schema.sql next to the executable” at deploy time. Nest/Next often ship assets beside the app; Go prefers embedding small assets.

### Pattern F — Explicit errors as values

```go
var ErrNotFound = errors.New("not found")
```

Handlers map `ErrNotFound` → 404, `ErrConflict` → 409. Go has no exceptions; you `return err` and branch with `errors.Is`. That feels verbose after Python, but it makes control flow obvious.

### Pattern G — Package-by-layer (for now)

Packages are named by **technical role** (`api`, `db`, `models`), not by feature (`jobs`, `applications`). That matches Phase 1 size. If the app grows, Go teams often switch to **package-by-feature** (`internal/jobs`, `internal/apply`). Either is fine; don’t over-split early.

---

## 4. Mental map: Nest / FastAPI → Go

| Concept you know | Here |
|---|---|
| `main.ts` / `uvicorn` entry | `cmd/server/main.go` |
| Nest module / FastAPI router | `internal/api/router.go` |
| Controller method | `handleListJobs`, etc. |
| Prisma / SQLAlchemy model | `internal/models` + `schema.sql` |
| Repository | `internal/db/store.go` |
| `.env` | `godotenv` in `main` |
| Swagger `/docs` | `/docs` + embedded OpenAPI |
| `pip install` / `package.json` | `go.mod` + `go.sum` |
| `src/` privacy by convention | `internal/` privacy by compiler |

---

## 5. How to run and explore

```bash
cd backend
go run ./cmd/server
```

| URL | Purpose |
|---|---|
| http://localhost:8000/docs | Interactive API docs (try requests) |
| http://localhost:8000/openapi.yaml | Raw OpenAPI 3 spec |
| http://localhost:8000/health | Liveness |
| http://localhost:8000/api/stats | Pipeline counts |

Tests that don’t need a live port:

```bash
go test ./internal/api/ -v
```

---

## 6. What comes next (same patterns)

| Phase | New package / idea |
|---|---|
| 2 Scrapers | `internal/scrapers` — each board is a type implementing a small `Scraper` interface |
| 3 Drafter / Telegram | `internal/services` — interfaces for LLM + bot, injected like `Store` |
| 4 Dashboard | Next.js talks to this API (CORS already allows `localhost:3000`) |
| 5 Submitter | another service; browser automation via `chromedp`/`rod` (Go), not Playwright |

When you add those, keep the rule: **handlers stay thin; side effects and SQL stay out of `main`.**

---

## 7. Short glossary

- **Module** (`go.mod`): the import path root (`github.com/mueedx/job-bot/backend`).
- **Package**: one folder of Go files sharing `package name`.
- **Binary**: output of building a `main` package.
- **Interface** (future): Go’s duck-typed contracts — great for swapping a fake store in tests.
- **CGO**: calling C from Go; we avoid it so builds stay portable.

If something in the tree still feels arbitrary, it is probably either (1) standard Go layout (`cmd`/`internal`), or (2) deliberate Phase 1 simplicity (Store instead of a full service layer). Both are intentional.

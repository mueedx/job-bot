# `data/` — local runtime state

Everything in this directory is **yours** and is gitignored. Nothing here is
committed except the `*.example` templates.

| Path | What it is |
|---|---|
| `jobs.db` | SQLite database. Created on first run. Delete it to start over. |
| `target_companies.yaml` | Company board slugs the scrapers ask for. Seeded from the `.example` on first run; edit freely. |
| `project_bank.yaml` | Your real projects, used to write cover letters. Copy it from `project_bank.yaml.example` first. |
| `*.example` | Templates that ship with the repo. |

## Setting up cover-letter drafting

The drafter refuses to invent anything, so it needs your own facts:

```bash
cp data/project_bank.yaml.example data/project_bank.yaml
# then edit data/project_bank.yaml
```

Without that file, drafting returns a clear error instead of a made-up letter.

## Where this directory is

The server resolves its data directory in this order:

1. `DATA_DIR`, when set (Docker Compose sets this to `/data`).
2. `./data` — repo root.
3. `../data` — when the server is started from `backend/`.

## Docker

`docker compose` bind-mounts the repo's `data/` directory to `/data`, so the
files above are the same ones the container reads and writes. On Linux, files
created by the container are owned by `root`; run `sudo chown -R "$USER" data`
if that gets in your way.

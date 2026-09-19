# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue. Use GitHub's private vulnerability
reporting on the repository's **Security → Report a vulnerability** tab:

https://github.com/mueedx/job-bot/security/advisories/new

Include steps to reproduce and the impact you believe it has. You can expect an
initial response within a few days.

## Scope and threat model

This app is designed to be run locally by one person. There is **no
authentication** on the API, and it is not built to be exposed to the internet.

- **Do not expose the API (port 8000) publicly.** It can read and modify your
  job data and may spend money on LLM calls. Keep it on localhost, or put it
  behind your own auth proxy.
- CORS only allows the configured dashboard origin (`DASHBOARD_URL` or
  `CORS_ALLOWED_ORIGINS`). That limits browser-based abuse — it is not access
  control.
- Secrets live in `.env`, which is gitignored and never baked into the Docker
  image. `.dockerignore` also strips `.env*`, `*.pdf` and `*.db` from build
  contexts.

### Sensitive things this project holds

| Thing | Where | Notes |
|---|---|---|
| `OPENAI_API_KEY` | `.env` | Sent to your configured `OPENAI_API_BASE_URL` |
| `TELEGRAM_BOT_TOKEN` | `.env` | Grants control of that bot |
| Resume PDFs | `resumes/` | Personal data; gitignored |
| `data/project_bank.yaml` | `data/` | Your work history; gitignored |
| `data/jobs.db` | `data/` | Your applications; gitignored |

If you have ever committed any of these by accident, rotate the credential —
deleting the file in a later commit does not remove it from history.

# `resumes/` — put your resume PDFs here

Drop 1–3 PDFs into this directory and the app picks them up. Nothing else is
required: the server finds this folder automatically (`RESUME_DIR` overrides it).

**These files are gitignored** — `*.pdf` is never committed.

## Naming

Jobs are routed to one of three tracks (`fullstack`, `blockchain`, `fde`), and
each track resolves to a file in this order:

| # | Rule | Example |
|---|---|---|
| 1 | Exact track name wins | `fullstack.pdf` |
| 2 | Name tokens are matched | `Jane_Doe_Fullstack.pdf` |
| 3 | `default.pdf` covers any track without its own file | `default.pdf` |
| 4 | A lone PDF that suggests no other track is used for every track | `My_Resume.pdf` |

Recognised tokens:

| Track | Tokens |
|---|---|
| `fullstack` | `fullstack`, `web`, `react`, `nestjs`, `nodejs`, `typescript`, `node js`, `full stack` |
| `blockchain` | `blockchain`, `web3`, `solidity`, `evm`, `crypto`, `defi`, `substrate`, `smart contract` |
| `fde` | `fde`, `ai`, `agentic`, `mcp`, `rag`, `forward deployed`, `ai engineer`, `solutions architect`, `customer engineer` |

Tokens match whole words, so `web3.pdf` is never read as `web`.

## Verify what the server sees

```bash
curl http://localhost:8000/api/resumes
```

That returns the directory in use, every PDF found, the file each track resolved
to, and any track listed under `missing` — those fall back to a
`resumes/<track>.pdf` placeholder until you add a file.

## Notes

- Docker mounts this directory read-only at `/app/resumes`.
- Stored paths are always `resumes/<file>.pdf`, so records stay valid whether the
  server runs locally or in a container.
- The (not yet implemented) submitter will attach the resolved file directly.

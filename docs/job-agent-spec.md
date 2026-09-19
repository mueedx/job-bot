# Job Applying Mechanism — Technical Specification & Execution Plan

**Project:** Autonomous Job Finder, Matcher & Application Engine  
**Status:** Approved Specification (All 18 Design Branches Settled)  
**Date:** 2026-09-05  

> Original planning document, lightly redacted for the public repository —
> personal paths, names and project names removed. Treat it as background
> context for *why* the code is shaped the way it is, not as an up-to-date
> description of what is built. See [../README.md](../README.md) for the current
> state and [architecture.md](architecture.md) for how the code is organised.

---

## 1. Executive Summary

This system is an intelligent, self-hosted job search and application engine built for a **Full-Stack Engineering (Next.js/NestJS)**, **Backend Blockchain (EVM/Solidity/Bittensor)**, and **Forward Deployed / AI Engineering (MCP/A2A/Agents)** background.

### Core Architecture Pillars:
1. **Multi-Source Ingestion:** Automated scrapers querying Greenhouse/Lever/Ashby APIs and Web3/Crypto boards. (LinkedIn and Wellfound session scraping was considered and rejected — see the boards section below.)
2. **Intelligent Router & Matcher:** Evaluates postings against compensation, remote/relocation criteria, and assigns one of **3 master resume tracks**.
3. **Tailored Drafter:** Generates custom, verified cover letters and dynamic ATS answers grounded in a real, editable project portfolio (`data/project_bank.yaml`).
4. **Staged Execution Engine:**
   - **Stage 1 (Calibration):** First applications reviewed and submitted via the local dashboard or Telegram Bot.
   - **Stage 2 (Autonomous Mode):** High-scoring matches auto-applied via Greenhouse/Lever direct APIs (capped per day).
5. **Control Center:** Local Web Dashboard (`http://localhost:3000`) + Telegram Bot.

---

## 2. System Architecture Diagram

```
                             ┌──────────────────────────────────────┐
                             │       Job Sources & Discovery         │
                             │  • Greenhouse / Lever / Ashby APIs   │
                             │  • CryptoJobsList / RemoteOK / YC    │
                             └──────────────────┬───────────────────┘
                                                │
                                                ▼
┌───────────────────────────────────────────────────────────────────────────────────┐
│ Backend Engine (Go / SQLite)                                                       │
│                                                                                   │
│  ┌─────────────────┐      ┌─────────────────┐      ┌───────────────────────────┐  │
│  │ Ingestion &     │ ───► │ Scoring & Track │ ───► │ Drafter (LLM Engine)      │  │
│  │ Deduplication   │      │ Router          │      │ • Cover letter generation │  │
│  └─────────────────┘      └────────┬────────┘      │ • ATS question generation │  │
│                                    │               └─────────────┬─────────────┘  │
│                                    ▼                             │                │
│                        ┌───────────────────────┐                 │                │
│                        │ Resume Asset Selector │                 │                │
│                        │ • fullstack.pdf       │                 │                │
│                        │ • blockchain.pdf      │                 │                │
│                        │ • fde.pdf             │                 │                │
│                        └───────────────────────┘                 │                │
│                                                                  ▼                │
│                                                    ┌───────────────────────────┐  │
│                                                    │ Submission Dispatcher     │  │
│                                                    │ • Direct API POST         │  │
│                                                    │ • Playwright Form Fill    │  │
│                                                    └───────────────────────────┘  │
└───────────────────────────────────────┬───────────────────────────────────────────┘
                                        │
                    ┌───────────────────┴───────────────────┐
                    ▼                                       ▼
┌───────────────────────────────────────┐   ┌───────────────────────────────────────┐
│ Next.js 15 Web Dashboard              │   │ Interactive Telegram Bot              │
│ (http://localhost:3000)               │   │ • Real-time alerts with match scores  │
│ • Kanban Pipeline Board               │   │ • Inline [Approve] / [Discard]        │
│ • Rich Job Card & JD Viewer           │   │ • Commands: /stats, /today, /status   │
│ • Cover Letter Markdown Editor        │   └───────────────────────────────────────┘
│ • One-Click Submit Button             │
└───────────────────────────────────────┘
```

---

## 3. Database Schema (SQLite: `jobs.db`)

```sql
CREATE TABLE IF NOT EXISTS jobs (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    source              TEXT NOT NULL,            -- 'greenhouse', 'lever', 'ashby', 'linkedin', 'wellfound', 'web3'
    source_id           TEXT NOT NULL,            -- Unique identifier from source
    url                 TEXT NOT NULL UNIQUE,
    title               TEXT NOT NULL,
    company             TEXT NOT NULL,
    location            TEXT,
    is_remote           BOOLEAN DEFAULT 1,
    is_relocation       BOOLEAN DEFAULT 0,
    salary_min          INTEGER,                  -- Monthly USD
    salary_max          INTEGER,                  -- Monthly USD
    description         TEXT NOT NULL,
    status              TEXT DEFAULT 'discovered',-- 'discovered', 'scored', 'queued', 'approved', 'applied', 'rejected', 'interview'
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS matches (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id              INTEGER NOT NULL UNIQUE,
    score               REAL NOT NULL,            -- 0.0 to 1.0 (e.g. 0.85)
    track               TEXT NOT NULL,            -- 'fullstack', 'blockchain', 'fde'
    matched_skills      TEXT,                     -- JSON Array: ["Next.js", "NestJS", "Prisma"]
    missing_skills      TEXT,                     -- JSON Array: ["GraphQL", "Kafka"]
    score_reasons       TEXT,                     -- 2-sentence rationale
    scored_at           DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS applications (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id              INTEGER NOT NULL UNIQUE,
    resume_path         TEXT NOT NULL,            -- 'resumes/fullstack.pdf', etc.
    cover_letter        TEXT NOT NULL,
    custom_qa           TEXT,                     -- JSON Object: {"Why this company?": "..."}
    submission_method   TEXT NOT NULL,            -- 'direct_api', 'playwright', 'manual'
    submitted_at        DATETIME,
    auto_applied        BOOLEAN DEFAULT 0,
    submission_status   TEXT DEFAULT 'pending',   -- 'pending', 'success', 'failed'
    response_status     TEXT DEFAULT 'no_reply',  -- 'no_reply', 'interview', 'rejected', 'offer'
    notes               TEXT,
    FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS submission_logs (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    application_id      INTEGER NOT NULL,
    event_type          TEXT NOT NULL,            -- 'form_filled', 'api_success', 'captcha_encountered', 'error'
    payload             TEXT,                     -- JSON error details or responses
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(application_id) REFERENCES applications(id) ON DELETE CASCADE
);
```

---

## 4. Resume Classification & Routing Rules

The matcher categorizes postings into one of 3 resume tracks:

| Track | Selected Static PDF | Trigger Keywords (Case-Insensitive) |
|---|---|---|
| **Blockchain** | `resumes/blockchain.pdf` | `Solidity`, `EVM`, `Smart Contracts`, `Web3`, `DeFi`, `Chainlink`, `Foundry`, `Hardhat`, `Rust/Substrate`, `Bittensor`, `dTAO`, `DApp`, `Indexer`, `Subgraphs` |
| **FDE / AI** | `resumes/fde.pdf` | `Forward Deployed`, `FDE`, `AI Engineer`, `Agentic`, `Agents`, `MCP`, `Model Context Protocol`, `A2A`, `LLM`, `Prompt Engineering`, `Solutions Architect`, `Customer Engineer`, `RAG` |
| **Full-Stack** (Default) | `resumes/fullstack.pdf` | `Next.js`, `React`, `NestJS`, `Node.js`, `TypeScript`, `Prisma`, `MySQL`, `PostgreSQL`, `REST API`, `Full Stack Developer`, `Frontend`, `Backend` |

*Tie-break rule:* If a role mentions both Web3 and AI (e.g. Bittensor agent subnets), route to **FDE / AI** if customer/architecture-facing, or **Blockchain** if protocol/smart-contract-heavy.

---

## 5. Candidate Project Bank (for Cover Letters & Screening Q&A)

The LLM drafter is anchored strictly to a real production portfolio (editable in
`data/project_bank.yaml`), so every claim can be traced back to a real project:

1. **Subnet.ai (Bittensor Ecosystem):**
   - End-to-end full-stack: Next.js frontend, NestJS/Prisma/MySQL backend (113,000+ LOC TypeScript).
   - Real-time chain data materialization via TaoStats: 9 BullMQ sync queues, 23 scheduled cron jobs.
   - Core dTAO emissions engine, Polkadot API wallet connect, TAO transfers/staking/swapping, Stripe premium checkout.
2. **3E (AI / Agentic Systems):**
   - Adopted internal AI tools, MCPs, and cloud agents delivering 44% engineering productivity gains.
   - Built an RFP AI management system for automated proposal drafting and review.
   - Designed Google Agent-to-Agent (A2A) protocol integration platforms and an internal agent marketplace.
3. **Loot8.io & Antematter (Web3 / Blockchain):**
   - End-to-end Next.js and backend microservices; improved app speed by 90% via indexers, Chainlink Functions, and GraphQL.
   - Scalability audits, DeFi smart contracts, and mainnet deployments across multiple EVM chains.
4. **Sahal Wallet (MRHB DeFi):**
   - World's first Shariah-compliant crypto wallet: integrated virtual prepaid cards, gift cards, and global mobile top-ups.

---

## 6. Sourcing & Application Dispatch Pipeline

### Sourcing Channels
- **Greenhouse / Lever / Ashby:** Direct API ingestion for 50+ curated high-growth AI and Web3 startup slugs (e.g. `langchain`, `replicate`, `together-ai`, `anysphere`, `polygon`, `chainlink`).
- **Web3 / Startup Portals:** CryptoJobsList, RemoteOK RSS/JSON, and Y Combinator Work at a Startup.
- **LinkedIn & Wellfound:** Scraped using session cookies (`li_at` for LinkedIn) with Playwright headless instances.

### Submission Dispatch Logic
- **Direct API:** Formats application payload with `resume_path` (multipart/form-data) and cover letter text; submits to Greenhouse `/v1/boards/{company}/jobs/{job_id}/applications`.
- **Playwright Automation:** Uses headless Chromium with injected session cookies. Auto-fills standard text inputs, selects resume file, and submits. If an unhandled CAPTCHA is detected, flags the job in Telegram and opens headed mode for manual completion.
- **Pacing & Safety:**
  - Strict limit: **5 to 10 applications per day**.
  - Randomized jitter: **45 to 90 seconds** delay between submissions.

---

## 7. User Interface Specifications

### Localhost Dashboard (`http://localhost:3000`)
- **Framework:** Next.js 15 App Router + Tailwind CSS + Lucide Icons.
- **Views:**
  1. **Kanban Pipeline:** Columns for `Discovered`, `High Match (Ready)`, `Applied`, `Interview`, `Archived`.
  2. **Job Detail Card:** Full JD viewer, match score breakdown, selected resume track badge, and interactive markdown cover letter editor.
  3. **Action Bar:** One-click `[Approve & Submit]`, `[Switch Resume Track]`, or `[Discard]`.
  4. **Analytics Page:** Breakdown of applications sent, response rates, and daily submission quotas.

### Interactive Telegram Bot
- **Bot Token & Chat ID:** Configured in `.env`.
- **Live Match Card:**
  ```
  🚀 New High Match (Score: 88%)
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Role: Senior Full-Stack Engineer (AI/Web3)
  Company: Replicate (Remote)
  Compensation: $3,500 - $5,000 / mo
  Track: Full-Stack (resumes/fullstack.pdf)
  
  Matched Skills: Next.js, NestJS, TypeScript, AI Agents
  
  [ 📄 View Draft ]  [ ✅ Approve & Submit ]  [ ❌ Discard ]
  ```
- **Bot Commands:**
  - `/stats` — Shows total discovered, applied, and interview count.
  - `/today` — Lists all jobs queued or applied today against the daily limit.
  - `/status` — Displays background scraper health and next scheduled run.

---

## 8. Directory Layout

```
job-agent/
├── backend/
│   ├── app/
│   │   ├── api/
│   │   │   ├── jobs.py             # Job CRUD & status endpoints
│   │   │   ├── apply.py            # Submission trigger endpoints
│   │   │   └── stats.py            # Analytics endpoints
│   │   ├── scrapers/
│   │   │   ├── greenhouse.py       # Greenhouse API board scraper
│   │   │   ├── lever.py            # Lever API board scraper
│   │   │   ├── remoteok.py         # RemoteOK API scraper
│   │   │   ├── cryptojobs.py       # Web3 scraper
│   │   │   └── playwright_base.py  # Headless browser scraper
│   │   ├── services/
│   │   │   ├── matcher.py          # Fit scoring & track classification
│   │   │   ├── drafter.py          # LLM cover letter & ATS answer generator
│   │   │   ├── submitter.py        # API + Playwright form submission
│   │   │   └── telegram_bot.py     # Interactive Telegram Bot runner
│   │   ├── models.py               # SQLAlchemy models
│   │   ├── database.py             # SQLite setup
│   │   └── main.py                 # FastAPI app entry point
│   ├── data/
│   │   ├── jobs.db                 # SQLite database
│   │   └── target_companies.yaml   # Curated list of Greenhouse/Lever company slugs
│   ├── resumes/
│   │   ├── fullstack.pdf           # Master Full-Stack Resume
│   │   ├── blockchain.pdf          # Master Backend Blockchain Resume
│   │   └── fde.pdf                 # Master Forward Deployed / AI Resume
│   ├── requirements.txt
│   └── .env
│
└── frontend/                       # Next.js 15 Dashboard
    ├── src/
    │   ├── app/
    │   │   ├── page.tsx            # Main Kanban board
    │   │   ├── jobs/[id]/page.tsx  # Detailed job review & editor
    │   │   └── stats/page.tsx      # Analytics dashboard
    │   ├── components/
    │   │   ├── JobCard.tsx
    │   │   ├── KanbanColumn.tsx
    │   │   └── CoverLetterEditor.tsx
    │   └── lib/
    │       └── api.ts              # FastAPI client
    ├── package.json
    └── tailwind.config.ts
```

---

## 9. Implementation Roadmap

| Phase | Component | Key Deliverables |
|---|---|---|
| **Phase 1: Foundation** | Project Scaffolding & Database | Directory structure, SQLite schema, FastAPI backend base, resume asset storage. |
| **Phase 2: Ingestion & Routing** | Scrapers & Matcher | Greenhouse/RemoteOK/Crypto scrapers, classification engine routing to the 3 resume PDFs, LLM fit scorer. |
| **Phase 3: Drafter & Telegram** | Tailoring & Mobile Alerting | LLM cover letter/ATS answer generator, interactive Telegram bot with inline action buttons. |
| **Phase 4: Dashboard** | Next.js 15 Web UI | Localhost Kanban interface, job detail view with live cover letter editor, and one-click submission triggers. |
| **Phase 5: Submitter Engine** | Direct API & Playwright | Direct Greenhouse/Lever POST submitter, Playwright browser form-filler with 5–10/day pacing and safety guards. |
| **Phase 6: End-to-End Verification** | Stage 1 Calibration | Live dry-run, Telegram notification testing, first 10 applications reviewed and verified. |

---

*Specification locked. Ready to execute Phase 1.*

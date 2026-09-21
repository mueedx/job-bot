CREATE TABLE IF NOT EXISTS jobs (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    source              TEXT NOT NULL,
    source_id           TEXT NOT NULL,
    url                 TEXT NOT NULL UNIQUE,
    title               TEXT NOT NULL,
    company             TEXT NOT NULL,
    location            TEXT,
    is_remote           BOOLEAN DEFAULT 1,
    is_relocation       BOOLEAN DEFAULT 0,
    salary_min          INTEGER,
    salary_max          INTEGER,
    description         TEXT NOT NULL,
    posted_at           DATETIME,
    status              TEXT DEFAULT 'discovered',
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    -- Eligibility rules (services.EligibilityRules): the stored verdict, the
    -- rule that decided it, a human-readable reason, the matched phrases as a
    -- JSON array, and whether the engine (rather than the operator) set status.
    eligibility         TEXT,
    eligibility_rule    TEXT,
    eligibility_reason  TEXT,
    eligibility_signals TEXT,
    eligibility_applied BOOLEAN DEFAULT 0
);

CREATE TABLE IF NOT EXISTS matches (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id              INTEGER NOT NULL UNIQUE,
    score               REAL NOT NULL,
    track               TEXT NOT NULL,
    matched_skills      TEXT,
    missing_skills      TEXT,
    score_reasons       TEXT,
    scored_at           DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS applications (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id              INTEGER NOT NULL UNIQUE,
    resume_path         TEXT NOT NULL,
    cover_letter        TEXT NOT NULL,
    custom_qa           TEXT,
    submission_method   TEXT NOT NULL,
    submitted_at        DATETIME,
    auto_applied        BOOLEAN DEFAULT 0,
    submission_status   TEXT DEFAULT 'pending',
    response_status     TEXT DEFAULT 'no_reply',
    notes               TEXT,
    FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS submission_logs (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    application_id      INTEGER NOT NULL,
    event_type          TEXT NOT NULL,
    payload             TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(application_id) REFERENCES applications(id) ON DELETE CASCADE
);

-- AI-extracted resume profiles (services.ResumeAnalyzer). One row per resume
-- file, keyed by its stored path; content_hash skips re-extraction when the
-- PDF has not changed, so each resume costs one LLM call per edit.
CREATE TABLE IF NOT EXISTS resume_profiles (
    resume_path    TEXT PRIMARY KEY,
    content_hash   TEXT NOT NULL,
    track          TEXT NOT NULL,
    skills         TEXT,
    keywords       TEXT,
    seniority      TEXT,
    summary        TEXT,
    extracted_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

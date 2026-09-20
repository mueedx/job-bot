const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8000";

/**
 * Track tokens mirrored from the backend (internal/resumes). Keep the two lists
 * in sync so the UI shows the same track the server picked for a resume file.
 */
const TRACK_TOKENS: Array<[string, string[]]> = [
  [
    "fde",
    [
      "fde",
      "ai",
      "agentic",
      "mcp",
      "rag",
      "forward deployed",
      "ai engineer",
      "solutions architect",
      "customer engineer",
    ],
  ],
  [
    "blockchain",
    [
      "blockchain",
      "web3",
      "solidity",
      "evm",
      "crypto",
      "defi",
      "substrate",
      "smart contract",
    ],
  ],
  [
    "fullstack",
    [
      "fullstack",
      "web",
      "react",
      "nestjs",
      "nodejs",
      "typescript",
      "node js",
      "full stack",
    ],
  ],
];

/** Base URL of the API, for links that point at the backend (e.g. /docs). */
export const API_BASE_URL = API_URL;

export type Job = {
  id: number;
  source: string;
  source_id: string;
  url: string;
  title: string;
  company: string;
  location: string | null;
  is_remote: boolean;
  is_relocation: boolean;
  salary_min: number | null;
  salary_max: number | null;
  description: string;
  posted_at: string | null;
  status: string;
  created_at: string;
  score?: number | null;
  track?: string | null;
};

export type Match = {
  id: number;
  job_id: number;
  score: number;
  track: string;
  matched_skills: string | null;
  missing_skills: string | null;
  score_reasons: string | null;
  scored_at: string;
};

export type Application = {
  id: number;
  job_id: number;
  resume_path: string;
  cover_letter: string;
  custom_qa: string | null;
  submission_method: string;
  submitted_at: string | null;
  auto_applied: boolean;
  submission_status: string;
  response_status: string;
  notes: string | null;
};

export type JobDetail = {
  job: Job;
  match: Match | null;
  application: Application | null;
};

export type Stats = {
  total_jobs: number;
  total_applications: number;
  applied: number;
  interviews: number;
  by_status: Record<string, number>;
  daily_limit: number;
};

export type IngestLogLine = {
  ts: string;
  level: "info" | "warn" | "error" | string;
  message: string;
};

export type IngestStatus = {
  running: boolean;
  ok: boolean;
  phase: string;
  current_source: string;
  sources_total: number;
  sources_done: number;
  last_started: string | null;
  last_finished: string | null;
  inserted: number;
  scored: number;
  drafted: number;
  skipped_old: number;
  source_errors: Record<string, string>;
  message: string;
  logs: IngestLogLine[];
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers ?? {}),
      },
      cache: "no-store",
    });
  } catch {
    throw new Error("Cannot reach API on :8000. Start Docker/API and try again.");
  }
  const text = await res.text();
  let body: unknown = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = { detail: text };
    }
  }
  if (!res.ok) {
    const detail =
      typeof body === "object" && body && "detail" in body
        ? String((body as { detail: unknown }).detail)
        : `Request failed (${res.status})`;
    const err = new Error(detail) as Error & { status: number; body: unknown };
    err.status = res.status;
    err.body = body;
    throw err;
  }
  return body as T;
}

export const api = {
  listJobsEnriched: () => request<Job[]>(`/api/jobs?enrich=1&limit=500`),
  getDetail: (id: number) => request<JobDetail>(`/api/jobs/${id}/detail`),
  saveApplication: (id: number, payload: { cover_letter?: string; track?: string }) =>
    request<Application>(`/api/jobs/${id}/application`, {
      method: "PUT",
      body: JSON.stringify(payload),
    }),
  updateJobStatus: (id: number, status: string) =>
    request<Job>(`/api/jobs/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    }),
  discard: (id: number) =>
    request<Job>(`/api/jobs/${id}/discard`, { method: "POST" }),
  approve: async (id: number) => {
    const res = await fetch(`${API_URL}/api/jobs/${id}/approve`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      cache: "no-store",
    });
    const body = await res.json();
    if (!res.ok) {
      const err = new Error(body?.detail ?? "Approve failed") as Error & {
        status: number;
        body: unknown;
      };
      err.status = res.status;
      err.body = body;
      throw err;
    }
    return body;
  },
  stats: () => request<Stats>(`/api/stats`),
  seed: () => request<{ created: number }>(`/api/dev/seed`, { method: "POST" }),
  startIngest: () =>
    request<IngestStatus>(`/api/ingest/run`, { method: "POST" }),
  ingestStatus: () => request<IngestStatus>(`/api/ingest/status`),
};

export function columnForStatus(status: string): string {
  switch (status) {
    case "discovered":
      return "discovered";
    case "scored":
    case "queued":
    case "approved":
      return "ready";
    case "applied":
      return "applied";
    case "interview":
      return "interview";
    case "rejected":
      return "archived";
    default:
      return "discovered";
  }
}

/**
 * Inverse of columnForStatus: the status to persist when a card is dropped on
 * a board column. "ready" stores `approved` because the board folds
 * scored/queued/approved into one High Match column.
 */
export function statusForColumn(column: string): string {
  switch (column) {
    case "ready":
      return "approved";
    case "applied":
      return "applied";
    case "interview":
      return "interview";
    case "archived":
      return "rejected";
    default:
      return "discovered";
  }
}

export function parseSkills(raw: string | null | undefined): string[] {
  if (!raw) return [];
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v.map(String) : [];
  } catch {
    return [];
  }
}

export function trackFromResumePath(path: string | null | undefined): string {
  if (!path) return "fullstack";
  const name = path.split("/").pop() ?? "";
  const normalized = name
    .replace(/\.pdf$/i, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, " ")
    .trim();
  const padded = ` ${normalized} `;

  let best = "fullstack";
  let bestScore = 0;
  for (const [track, tokens] of TRACK_TOKENS) {
    let score = 0;
    for (const token of tokens) {
      if (padded.includes(` ${token} `)) score += token.includes(" ") ? 2 : 1;
    }
    if (score > bestScore) {
      best = track;
      bestScore = score;
    }
  }
  return best;
}

/** Human-readable posting age, or null when the source gave no date. */
export function formatPostAge(
  postedAt: string | null | undefined,
): string | null {
  if (!postedAt) return null;
  const d = new Date(postedAt);
  if (Number.isNaN(d.getTime())) return null;
  const days = Math.floor((Date.now() - d.getTime()) / 86_400_000);
  if (days <= 0) return "today";
  if (days === 1) return "1d ago";
  if (days < 30) return `${days}d ago`;
  return `${Math.floor(days / 30)}mo ago`;
}

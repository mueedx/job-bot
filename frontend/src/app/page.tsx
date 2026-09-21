"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ActiveSources } from "@/components/ActiveSources";
import { IngestPanel } from "@/components/IngestPanel";
import { KanbanColumn } from "@/components/KanbanColumn";
import {
  api,
  columnForStatus,
  statusForColumn,
  type Job,
} from "@/lib/api";

const COLUMNS = [
  { key: "discovered", title: "Discovered" },
  { key: "ready", title: "High Match (Ready)" },
  { key: "applied", title: "Applied" },
  { key: "interview", title: "Interview" },
  { key: "archived", title: "Archived" },
] as const;

export default function PipelinePage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [seeding, setSeeding] = useState(false);
  // Confidence filter: 0 shows everything. Kept in localStorage so the board
  // opens the way you left it. Unscored jobs are never hidden by it.
  const [minScore, setMinScore] = useState(0);

  useEffect(() => {
    const raw = window.localStorage.getItem("jobbot:min-score");
    const n = raw == null ? 0 : Number(raw);
    if (Number.isFinite(n) && n >= 0 && n <= 1) setMinScore(n);
  }, []);

  function changeMinScore(value: number) {
    setMinScore(value);
    window.localStorage.setItem("jobbot:min-score", String(value));
  }

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.listJobsEnriched();
      setJobs(data);
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "Cannot reach API. Is the Go server on :8000?",
      );
      setJobs([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const visible = useMemo(
    () =>
      minScore > 0
        ? jobs.filter((j) => j.score == null || j.score >= minScore)
        : jobs,
    [jobs, minScore],
  );
  const hiddenCount = jobs.length - visible.length;

  const grouped = useMemo(() => {
    const map: Record<string, Job[]> = {
      discovered: [],
      ready: [],
      applied: [],
      interview: [],
      archived: [],
    };
    for (const job of visible) {
      const col = columnForStatus(job.status);
      (map[col] ?? map.discovered).push(job);
    }
    return map;
  }, [visible]);

  /**
   * Optimistically move a card to another column, then persist the status via
   * PATCH /api/jobs/{id}. On failure the card goes back exactly where it was
   * and the error banner explains why — the board never shows a state the
   * server did not accept.
   */
  async function handleMove(jobId: number, toColumn: string) {
    const snapshot = jobs;
    const status = statusForColumn(toColumn);
    setJobs(jobs.map((j) => (j.id === jobId ? { ...j, status } : j)));
    try {
      await api.updateJobStatus(jobId, status);
    } catch (e) {
      setJobs(snapshot);
      setError(
        e instanceof Error
          ? `Move failed: ${e.message}`
          : "Move failed. Cannot reach the API.",
      );
    }
  }

  async function seed() {
    setSeeding(true);
    try {
      await api.seed();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Seed failed");
    } finally {
      setSeeding(false);
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-2xl font-medium tracking-tight">Pipeline</h2>
          <p className="mt-1 max-w-xl text-sm text-[var(--muted)]">
            Stage 1 calibration board. Open a role to review drafts before any
            real submission.
          </p>
        </div>
        <IngestPanel
          onComplete={load}
          actions={
            <>
              <button
                type="button"
                onClick={() => void load()}
                className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
              >
                Refresh
              </button>
              <button
                type="button"
                disabled={seeding}
                onClick={() => void seed()}
                className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm disabled:opacity-60"
              >
                {seeding ? "Seeding…" : "Load demo data"}
              </button>
            </>
          }
        />
      </div>

      <ActiveSources />

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <label className="flex items-center gap-2 text-[var(--muted)]">
          Min confidence
          <select
            value={String(minScore)}
            onChange={(e) => changeMinScore(Number(e.target.value))}
            className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-2 py-1.5 text-sm text-[var(--ink)]"
          >
            <option value="0">Any</option>
            <option value="0.5">50%+</option>
            <option value="0.6">60%+</option>
            <option value="0.7">70%+</option>
            <option value="0.8">80%+</option>
            <option value="0.9">90%+</option>
          </select>
        </label>
        {hiddenCount > 0 ? (
          <span className="text-xs text-[var(--muted)]">
            {hiddenCount} job{hiddenCount === 1 ? "" : "s"} below the threshold
            hidden
          </span>
        ) : null}
      </div>

      {error ? (
        <div
          className="rounded-sm border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
          role="alert"
        >
          {error}
        </div>
      ) : null}

      {loading ? (
        <p className="text-sm text-[var(--muted)]">Loading pipeline…</p>
      ) : (
        <div className="flex gap-4 overflow-x-auto pb-2">
          {COLUMNS.map((col, i) => (
            <KanbanColumn
              key={col.key}
              title={col.title}
              column={col.key}
              jobs={grouped[col.key] ?? []}
              delayMs={i * 40}
              onMoveJob={(jobId, target) => void handleMove(jobId, target)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

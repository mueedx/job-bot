"use client";

import { useState } from "react";
import { columnForStatus, type Job } from "@/lib/api";
import { JobCard } from "./JobCard";

export function KanbanColumn({
  title,
  column,
  jobs,
  delayMs = 0,
  onMoveJob,
}: {
  title: string;
  column: string;
  jobs: Job[];
  delayMs?: number;
  onMoveJob?: (jobId: number, toColumn: string) => void;
}) {
  const [dropTarget, setDropTarget] = useState(false);

  function handleDragOver(e: React.DragEvent) {
    if (!e.dataTransfer.types.includes("text/job-id")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDropTarget(true);
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDropTarget(false);
    const id = Number(e.dataTransfer.getData("text/job-id"));
    if (!Number.isFinite(id) || id <= 0 || !onMoveJob) return;
    const from = jobs.find((j) => j.id === id);
    if (from && columnForStatus(from.status) === column) return; // no-op drop
    onMoveJob(id, column);
  }

  return (
    <section
      onDragOver={handleDragOver}
      onDragLeave={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node)) {
          setDropTarget(false);
        }
      }}
      onDrop={handleDrop}
      className={`fade-up flex min-w-[240px] flex-1 flex-col gap-3 rounded-sm border border-transparent px-1 py-1 transition-colors ${
        dropTarget
          ? "border-[var(--accent)] bg-[var(--accent-soft)]"
          : ""
      }`}
      style={{ animationDelay: `${delayMs}ms` }}
    >
      <header className="flex items-baseline justify-between gap-2 border-b border-[var(--line)] pb-2">
        <h2 className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
          {title}
        </h2>
        <span className="mono text-[11px] text-[var(--ink)]">{jobs.length}</span>
      </header>
      <div className="flex flex-col gap-2">
        {jobs.length === 0 ? (
          <p className="rounded-sm border border-dashed border-[var(--line)] px-3 py-6 text-center text-xs text-[var(--muted)]">
            Empty
          </p>
        ) : (
          jobs.map((job) => <JobCard key={job.id} job={job} />)
        )}
      </div>
    </section>
  );
}

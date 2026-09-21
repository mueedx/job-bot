"use client";

import Link from "next/link";
import { useState } from "react";
import { Lock } from "lucide-react";
import { formatPostAge, eligibilitySummary, type Job } from "@/lib/api";
import { sourceNote } from "@/lib/sourceNotes";

export function JobCard({ job }: { job: Job }) {
  const score =
    job.score != null ? `${Math.round(job.score * 100)}%` : null;
  const age = formatPostAge(job.posted_at);
  const note = sourceNote(job.source);
  const elig = eligibilitySummary(job);
  const [dragging, setDragging] = useState(false);

  return (
    <Link
      href={`/jobs/${job.id}`}
      draggable
      onDragStart={(e) => {
        // Native DnD: carry the job id so the column's onDrop can move it.
        e.dataTransfer.setData("text/job-id", String(job.id));
        e.dataTransfer.effectAllowed = "move";
        setDragging(true);
      }}
      onDragEnd={() => setDragging(false)}
      className={`block cursor-grab rounded-sm border border-[var(--line)] bg-[var(--panel)] p-3 transition-transform duration-150 hover:-translate-y-0.5 hover:border-[var(--accent)] active:cursor-grabbing ${
        dragging ? "opacity-50" : ""
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="text-sm font-medium leading-snug text-[var(--ink)]">
            {job.title}
          </p>
          <p className="mt-1 text-xs text-[var(--muted)]">{job.company}</p>
        </div>
        {score ? (
          <span className="mono shrink-0 rounded-sm bg-[var(--accent-soft)] px-1.5 py-0.5 text-[11px] text-[var(--accent)]">
            {score}
          </span>
        ) : null}
      </div>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {job.track ? (
          <span className="mono rounded-sm border border-[var(--line)] px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-[var(--muted)]">
            {job.track}
          </span>
        ) : null}
        {job.is_remote ? (
          <span className="mono rounded-sm border border-[var(--line)] px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-[var(--muted)]">
            remote
          </span>
        ) : null}
        {age ? (
          <span className="mono rounded-sm border border-[var(--line)] px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-[var(--muted)]">
            {age}
          </span>
        ) : null}
        {elig.status === "veto" ? (
          <span
            title={elig.reason || "Rejected by your eligibility rules"}
            className="mono cursor-help rounded-sm border border-red-300 bg-red-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-red-800 dark:border-red-500/40 dark:bg-red-500/15 dark:text-red-300"
          >
            vetoed
          </span>
        ) : null}
        {elig.status === "pass" ? (
          <span
            title={elig.reason || "Passed your eligibility rules"}
            className="mono cursor-help rounded-sm border border-emerald-300 bg-emerald-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-emerald-800 dark:border-emerald-500/40 dark:bg-emerald-500/15 dark:text-emerald-300"
          >
            eligible
          </span>
        ) : null}
        {note ? (
          <span
            title={note.text}
            className="mono inline-flex cursor-help items-center gap-1 rounded-sm border border-amber-300 bg-amber-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-amber-800 dark:border-amber-500/40 dark:bg-amber-500/15 dark:text-amber-300"
          >
            <Lock className="h-3 w-3" aria-hidden="true" />
            {note.label}
          </span>
        ) : null}
      </div>
    </Link>
  );
}

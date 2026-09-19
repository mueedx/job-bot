"use client";

import Link from "next/link";
import { useState } from "react";
import { Lock } from "lucide-react";
import { formatPostAge, paywallNotice, type Job } from "@/lib/api";

export function JobCard({ job }: { job: Job }) {
  const score =
    job.score != null ? `${Math.round(job.score * 100)}%` : null;
  const age = formatPostAge(job.posted_at);
  const paywall = paywallNotice(job.source);
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
        {paywall ? (
          <span
            title={paywall}
            className="mono inline-flex cursor-help items-center gap-1 rounded-sm border border-amber-300 bg-amber-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-amber-800 dark:border-amber-500/40 dark:bg-amber-500/15 dark:text-amber-300"
          >
            <Lock className="h-3 w-3" aria-hidden="true" />
            premium
          </span>
        ) : null}
      </div>
    </Link>
  );
}

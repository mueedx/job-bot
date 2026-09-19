"use client";

import { useEffect, useState } from "react";
import { api, type Stats } from "@/lib/api";

export default function StatsPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setStats(await api.stats());
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load stats");
      }
    })();
  }, []);

  if (error) {
    return (
      <p className="text-sm text-red-700" role="alert">
        {error}
      </p>
    );
  }

  if (!stats) {
    return <p className="text-sm text-[var(--muted)]">Loading analytics…</p>;
  }

  const discovered = stats.by_status.discovered ?? 0;
  const cards = [
    { label: "Total jobs", value: stats.total_jobs },
    { label: "Discovered", value: discovered },
    { label: "Applied", value: stats.applied },
    { label: "Interviews", value: stats.interviews },
    { label: "Applications rows", value: stats.total_applications },
    { label: "Daily limit (guidance)", value: stats.daily_limit },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-medium tracking-tight">Analytics</h2>
        <p className="mt-1 text-sm text-[var(--muted)]">
          Stage 1 pacing snapshot. Daily limit is configured guidance until the
          submitter records real sends.
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {cards.map((c) => (
          <div
            key={c.label}
            className="rounded-sm border border-[var(--line)] bg-[var(--panel)] p-4"
          >
            <p className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
              {c.label}
            </p>
            <p className="mt-2 text-3xl font-medium text-[var(--ink)]">
              {c.value}
            </p>
          </div>
        ))}
      </div>

      <section className="rounded-sm border border-[var(--line)] bg-[var(--panel)] p-4">
        <h3 className="mono mb-3 text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
          By status
        </h3>
        <ul className="space-y-2">
          {Object.keys(stats.by_status).length === 0 ? (
            <li className="text-sm text-[var(--muted)]">No jobs yet.</li>
          ) : (
            Object.entries(stats.by_status).map(([status, count]) => (
              <li
                key={status}
                className="flex items-center justify-between border-b border-[var(--line)] py-2 text-sm last:border-0"
              >
                <span className="mono uppercase tracking-wide text-[var(--muted)]">
                  {status}
                </span>
                <span className="font-medium">{count}</span>
              </li>
            ))
          )}
        </ul>
      </section>
    </div>
  );
}

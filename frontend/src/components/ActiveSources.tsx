"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, type SettingsResponse } from "@/lib/api";

/**
 * Compact "what will a search actually use" strip for the pipeline header:
 * enabled sources and target regions, linking to /settings to change them.
 */
export function ActiveSources() {
  const [data, setData] = useState<SettingsResponse | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .getSettings()
      .then((d) => {
        if (!cancelled) setData(d);
      })
      .catch(() => {
        /* the strip is informational; the page still works without it */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (!data) return null;
  const active = data.sources.filter((s) => s.enabled && s.ready);
  if (active.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
        Active
      </span>
      {active.map((s) => (
        <span
          key={s.name}
          title={s.note ?? s.label}
          className="mono rounded-sm border border-[var(--line)] bg-[var(--panel)] px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-[var(--muted)]"
        >
          {s.label}
        </span>
      ))}
      {data.settings.recruiter_countries.map((c) => (
        <span
          key={c}
          className="mono rounded-sm border border-[var(--accent)] bg-[var(--accent-soft)] px-1.5 py-0.5 text-[10px] uppercase text-[var(--accent)]"
        >
          {c}
        </span>
      ))}
      <Link
        href="/settings"
        className="mono ml-1 text-[11px] text-[var(--accent)] underline underline-offset-2"
      >
        Change
      </Link>
    </div>
  );
}

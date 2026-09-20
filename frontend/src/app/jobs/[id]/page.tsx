"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ExternalLink, Lock } from "lucide-react";
import { ActionBar } from "@/components/ActionBar";
import { CoverLetterEditor } from "@/components/CoverLetterEditor";
import {
  api,
  parseSkills,
  trackFromResumePath,
  type JobDetail,
} from "@/lib/api";
import { sourceNote } from "@/lib/sourceNotes";

const TRACKS = ["fullstack", "blockchain", "fde"] as const;

export default function JobDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = Number(params.id);
  const [detail, setDetail] = useState<JobDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await api.getDetail(id);
      setDetail(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load job");
    }
  }, [id]);

  useEffect(() => {
    if (!Number.isFinite(id)) {
      setError("Invalid job id");
      return;
    }
    void load();
  }, [id, load]);

  const track = useMemo(() => {
    if (detail?.match?.track) return detail.match.track;
    return trackFromResumePath(detail?.application?.resume_path);
  }, [detail]);

  async function saveLetter(cover_letter: string) {
    await api.saveApplication(id, { cover_letter });
    await load();
  }

  async function switchTrack() {
    const idx = TRACKS.indexOf(track as (typeof TRACKS)[number]);
    const next = TRACKS[(idx + 1) % TRACKS.length];
    setBusy(true);
    setNotice(null);
    try {
      await api.saveApplication(id, { track: next });
      await load();
      setNotice(`Resume track set to ${next}`);
    } catch (e) {
      setNotice(e instanceof Error ? e.message : "Track switch failed");
    } finally {
      setBusy(false);
    }
  }

  async function discard() {
    setBusy(true);
    setNotice(null);
    try {
      await api.discard(id);
      router.push("/");
    } catch (e) {
      setNotice(e instanceof Error ? e.message : "Discard failed");
      setBusy(false);
    }
  }

  async function approve() {
    setBusy(true);
    setNotice(null);
    try {
      await api.approve(id);
      setNotice("Unexpected success — refresh to verify status.");
      await load();
    } catch (e) {
      const err = e as Error & { status?: number; body?: { detail?: string } };
      const detailMsg = err.body?.detail ?? err.message;
      setNotice(detailMsg);
      await load();
    } finally {
      setBusy(false);
    }
  }

  if (error && !detail) {
    return (
      <div className="space-y-3">
        <Link href="/" className="text-sm text-[var(--accent)]">
          ← Pipeline
        </Link>
        <p className="text-sm text-red-700" role="alert">
          {error}
        </p>
      </div>
    );
  }

  if (!detail) {
    return <p className="text-sm text-[var(--muted)]">Loading role…</p>;
  }

  const { job, match, application } = detail;
  const matched = parseSkills(match?.matched_skills);
  const missing = parseSkills(match?.missing_skills);
  const note = sourceNote(job.source);

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <Link href="/" className="text-sm text-[var(--accent)]">
        ← Pipeline
      </Link>

      <header className="space-y-2">
        <p className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
          {job.company} · {job.status}
        </p>
        <h2 className="text-3xl font-medium tracking-tight">{job.title}</h2>
        <div className="flex flex-wrap items-center gap-2 text-xs">
          {job.location ? (
            <span className="text-[var(--muted)]">{job.location}</span>
          ) : null}
          {job.salary_min != null && job.salary_max != null ? (
            <span className="text-[var(--muted)]">
              ${job.salary_min}–${job.salary_max}/mo
            </span>
          ) : null}
          {job.url ? (
            <a
              href={job.url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 font-medium text-[var(--accent)] underline underline-offset-2 hover:text-[var(--ink)]"
              title={job.url}
            >
              Source listing
              <ExternalLink className="h-3 w-3" aria-hidden="true" />
            </a>
          ) : (
            <span className="text-[var(--muted)]">No source link</span>
          )}
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
      </header>

      {notice ? (
        <div
          className="rounded-sm border border-[var(--line)] bg-[var(--accent-soft)] px-4 py-3 text-sm text-[var(--ink)]"
          role="status"
        >
          {notice}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-[1.2fr_0.8fr]">
        <div className="rounded-sm border border-[var(--line)] bg-[var(--panel)] p-4">
          <h3 className="mono mb-3 text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
            Job description
          </h3>
          <p className="whitespace-pre-wrap text-sm leading-relaxed text-[var(--ink)]">
            {job.description}
          </p>
        </div>

        <div className="space-y-4">
          <div className="rounded-sm border border-[var(--line)] bg-[var(--panel)] p-4">
            <h3 className="mono mb-3 text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
              Match
            </h3>
            {match ? (
              <div className="space-y-3 text-sm">
                <p className="text-2xl font-medium text-[var(--accent)]">
                  {Math.round(match.score * 100)}%
                </p>
                <p className="mono text-[11px] uppercase tracking-wide text-[var(--muted)]">
                  Track · {match.track}
                </p>
                {match.score_reasons ? (
                  <p className="text-[var(--ink)]">{match.score_reasons}</p>
                ) : null}
                {matched.length ? (
                  <div>
                    <p className="mono text-[10px] uppercase text-[var(--muted)]">
                      Matched
                    </p>
                    <p>{matched.join(", ")}</p>
                  </div>
                ) : null}
                {missing.length ? (
                  <div>
                    <p className="mono text-[10px] uppercase text-[var(--muted)]">
                      Missing
                    </p>
                    <p>{missing.join(", ")}</p>
                  </div>
                ) : null}
              </div>
            ) : (
              <p className="text-sm text-[var(--muted)]">No match data yet.</p>
            )}
          </div>
        </div>
      </section>

      <section className="rounded-sm border border-[var(--line)] bg-[var(--panel)] p-4">
        <CoverLetterEditor
          key={application?.id ?? "new"}
          initial={application?.cover_letter ?? ""}
          onSave={saveLetter}
        />
      </section>

      <ActionBar
        busy={busy}
        trackLabel={track}
        onApprove={() => void approve()}
        onDiscard={() => void discard()}
        onSwitchTrack={() => void switchTrack()}
      />
    </div>
  );
}

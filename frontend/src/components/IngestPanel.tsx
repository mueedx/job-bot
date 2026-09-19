"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { api, type IngestStatus } from "@/lib/api";

function phaseLabel(st: IngestStatus): string {
  switch (st.phase) {
    case "loading_targets":
      return "Loading company boards…";
    case "fetching":
      return st.current_source
        ? `Fetching ${prettySource(st.current_source)}…`
        : "Fetching boards…";
    case "matching":
      return "Matching roles…";
    case "drafting":
      return "Drafting cover letters…";
    case "done":
      return "Search complete";
    case "error":
      return "Search failed";
    default:
      return st.message || "Idle";
  }
}

function prettySource(name: string): string {
  switch (name) {
    case "greenhouse":
      return "Greenhouse";
    case "lever":
      return "Lever";
    case "ashby":
      return "Ashby";
    case "remoteok":
      return "RemoteOK";
    case "cryptojobs":
      return "Crypto / Web3";
    default:
      return name;
  }
}

type Props = {
  onComplete: () => void | Promise<void>;
  actions?: ReactNode;
};

export function IngestPanel({ onComplete, actions }: Props) {
  const [status, setStatus] = useState<IngestStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [banner, setBanner] = useState<{
    kind: "success" | "warn" | "error" | "info";
    text: string;
  } | null>(null);
  const [panelOpen, setPanelOpen] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const completedRef = useRef(false);
  const logEndRef = useRef<HTMLDivElement | null>(null);

  const stopPoll = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const applyStatus = useCallback(
    async (st: IngestStatus, opts?: { fromStart?: boolean }) => {
      setStatus(st);
      setPanelOpen(true);
      setBusy(st.running);

      if (st.running) {
        completedRef.current = false;
        setBanner(null);
        return;
      }

      if (completedRef.current) return;
      completedRef.current = true;
      stopPoll();

      if (!st.ok || st.phase === "error") {
        setBanner({
          kind: "error",
          text: st.message || "Search failed. Check the log below.",
        });
        return;
      }

      const errs = Object.keys(st.source_errors ?? {}).length;
      setBanner({
        kind: errs > 0 ? "warn" : "success",
        text:
          st.message ||
          `Found ${st.inserted} new roles (scored ${st.scored}).`,
      });
      if (!opts?.fromStart) {
        await onComplete();
      }
    },
    [onComplete, stopPoll],
  );

  const pollOnce = useCallback(async () => {
    try {
      const st = await api.ingestStatus();
      await applyStatus(st);
    } catch (e) {
      stopPoll();
      setBusy(false);
      setBanner({
        kind: "error",
        text:
          e instanceof Error && /fetch|network|failed/i.test(e.message)
            ? "Cannot reach API on :8000. Start Docker/API and try again."
            : e instanceof Error
              ? e.message
              : "Cannot reach API on :8000",
      });
      setPanelOpen(true);
    }
  }, [applyStatus, stopPoll]);

  const startPolling = useCallback(() => {
    stopPoll();
    void pollOnce();
    pollRef.current = setInterval(() => {
      void pollOnce();
    }, 1000);
  }, [pollOnce, stopPoll]);

  useEffect(() => {
    return () => stopPoll();
  }, [stopPoll]);

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, [status?.logs?.length]);

  async function startSearch() {
    setBanner(null);
    setPanelOpen(true);
    completedRef.current = false;
    setBusy(true);
    try {
      const st = await api.startIngest();
      await applyStatus(st, { fromStart: true });
      startPolling();
    } catch (e) {
      const err = e as Error & { status?: number; body?: { status?: IngestStatus } };
      if (err.status === 409) {
        setBanner({
          kind: "info",
          text: "Search already in progress",
        });
        if (err.body?.status) {
          await applyStatus(err.body.status, { fromStart: true });
        }
        startPolling();
        return;
      }
      setBusy(false);
      const msg =
        err instanceof Error
          ? /fetch|network|failed to fetch/i.test(err.message)
            ? "Cannot reach API on :8000. Start Docker/API and try again."
            : err.message
          : "Cannot reach API on :8000";
      setBanner({ kind: "error", text: msg });
    }
  }

  const running = busy || status?.running === true;
  const total = status?.sources_total ?? 0;
  const done = status?.sources_done ?? 0;
  const pct = total > 0 ? Math.min(100, Math.round((done / total) * 100)) : running ? 8 : 0;
  const sourceWarns = Object.entries(status?.source_errors ?? {});

  return (
    <div className="w-full space-y-2 sm:w-auto sm:min-w-[22rem] sm:max-w-lg">
      <div className="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          disabled={running}
          onClick={() => void startSearch()}
          className="rounded-sm bg-[var(--accent)] px-3 py-2 text-sm text-white disabled:opacity-60"
        >
          {running ? "Searching…" : "Search jobs"}
        </button>
        {actions}
      </div>

      {panelOpen ? (
        <div className="border border-[var(--line)] bg-[var(--panel)] p-3 text-left">
          {banner ? (
            <div
              className={
                banner.kind === "success"
                  ? "mb-2 border border-teal-200 bg-teal-50 px-2.5 py-1.5 text-sm text-teal-900"
                  : banner.kind === "warn"
                    ? "mb-2 border border-amber-200 bg-amber-50 px-2.5 py-1.5 text-sm text-amber-900"
                    : banner.kind === "info"
                      ? "mb-2 border border-[var(--line)] bg-[var(--wash)] px-2.5 py-1.5 text-sm text-[var(--ink)]"
                      : "mb-2 border border-red-200 bg-red-50 px-2.5 py-1.5 text-sm text-red-800"
              }
              role="status"
            >
              {banner.text}
            </div>
          ) : null}

          <div className="flex items-baseline justify-between gap-2 text-sm">
            <span className="text-[var(--ink)]">{status ? phaseLabel(status) : "Ready"}</span>
            <span className="mono text-xs text-[var(--muted)]">
              {total > 0 ? `${done}/${total}` : running ? "…" : "—"}
            </span>
          </div>
          <div className="mt-2 h-1.5 overflow-hidden bg-[var(--wash)]">
            <div
              className="h-full bg-[var(--accent)] transition-[width] duration-300 ease-out"
              style={{ width: `${running && pct < 5 ? 5 : pct}%` }}
            />
          </div>

          {sourceWarns.length > 0 ? (
            <ul className="mt-2 space-y-1 text-xs text-amber-800">
              {sourceWarns.map(([src, raw]) => (
                <li key={src}>
                  {prettySource(src)}: board issue (
                  {raw.length > 80 ? `${raw.slice(0, 80)}…` : raw})
                </li>
              ))}
            </ul>
          ) : null}

          <div className="mt-3 max-h-40 overflow-y-auto border border-[var(--line)] bg-[var(--wash)] px-2 py-1.5">
            <ul className="mono space-y-1 text-[11px] leading-snug text-[var(--ink)]">
              {(status?.logs ?? []).map((line, i) => (
                <li
                  key={`${line.ts}-${i}`}
                  className={
                    line.level === "error"
                      ? "text-red-700"
                      : line.level === "warn"
                        ? "text-amber-800"
                        : ""
                  }
                >
                  <span className="text-[var(--muted)]">{formatTs(line.ts)} </span>
                  {line.message}
                </li>
              ))}
              {(status?.logs?.length ?? 0) === 0 ? (
                <li className="text-[var(--muted)]">No log lines yet.</li>
              ) : null}
              <div ref={logEndRef} />
            </ul>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function formatTs(ts: string): string {
  try {
    const d = new Date(ts);
    if (Number.isNaN(d.getTime())) return "";
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  } catch {
    return "";
  }
}

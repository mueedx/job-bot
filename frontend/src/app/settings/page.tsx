"use client";

import { useCallback, useEffect, useState } from "react";
import {
  api,
  type SettingsResponse,
  type SourceHealth,
  type SourceInfo,
} from "@/lib/api";

const KIND_LABEL: Record<string, string> = {
  board: "company boards",
  aggregator: "aggregator API",
  feed: "public feed",
  gated: "opt-in (restricted)",
};

export default function SettingsPage() {
  const [data, setData] = useState<SettingsResponse | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [newRegion, setNewRegion] = useState("");
  const [health, setHealth] = useState<Record<string, SourceHealth> | null>(
    null,
  );
  const [checking, setChecking] = useState(false);

  const load = useCallback(async () => {
    try {
      setData(await api.getSettings());
      setLoadError(null);
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : "Cannot reach API on :8000");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (loadError && !data) {
    return (
      <div className="space-y-3">
        <h2 className="text-2xl font-medium tracking-tight">Settings</h2>
        <div
          className="rounded-sm border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
          role="alert"
        >
          {loadError}
        </div>
      </div>
    );
  }
  if (!data) {
    return <p className="text-sm text-[var(--muted)]">Loading settings…</p>;
  }

  const { settings } = data;
  const sources = [...data.sources].sort((a, b) =>
    a.label.localeCompare(b.label),
  );
  const enabledCount = sources.filter((s) => s.enabled && s.ready).length;

  function toggle(name: string) {
    setData((d) => {
      if (!d) return d;
      const current = d.settings.sources[name] ?? true;
      return {
        ...d,
        settings: {
          ...d.settings,
          sources: { ...d.settings.sources, [name]: !current },
        },
      };
    });
    setNotice(null);
  }

  function removeRegion(code: string) {
    setData((d) => {
      if (!d) return d;
      return {
        ...d,
        settings: {
          ...d.settings,
          recruiter_countries: d.settings.recruiter_countries.filter(
            (c) => c !== code,
          ),
        },
      };
    });
    setNotice(null);
  }

  function addRegion(code: string) {
    const normalized = code.trim().toLowerCase();
    if (!/^[a-z]{2}$/.test(normalized)) {
      setError("Region must be a 2-letter country code, e.g. ie, gb, ae.");
      return;
    }
    setData((d) => {
      if (!d) return d;
      if (d.settings.recruiter_countries.includes(normalized)) return d;
      return {
        ...d,
        settings: {
          ...d.settings,
          recruiter_countries: [...d.settings.recruiter_countries, normalized],
        },
      };
    });
    setNewRegion("");
    setError(null);
  }

  async function save() {
    setSaving(true);
    setError(null);
    setNotice(null);
    try {
      const saved = await api.saveSettings({
        sources: settings.sources,
        recruiter_countries: settings.recruiter_countries,
      });
      setData(saved);
      setNotice("Settings saved — the next search uses them.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  async function testSources() {
    setChecking(true);
    setError(null);
    try {
      const res = await api.sourcesHealth();
      const map: Record<string, SourceHealth> = {};
      for (const r of res.results) map[r.name] = r;
      setHealth(map);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Health check failed");
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-2xl font-medium tracking-tight">Settings</h2>
          <p className="mt-1 max-w-xl text-sm text-[var(--muted)]">
            Choose which job sites are searched and which regions to target.
            Saved to <span className="mono">data/settings.yaml</span> and applied
            to the next search.
          </p>
        </div>
        <button
          type="button"
          disabled={saving}
          onClick={() => void save()}
          className="rounded-sm border border-[var(--accent)] bg-[var(--accent-soft)] px-4 py-2 text-sm font-medium text-[var(--accent)] disabled:opacity-60"
        >
          {saving ? "Saving…" : "Save changes"}
        </button>
      </div>

      {notice ? (
        <div
          className="rounded-sm border border-[var(--line)] bg-[var(--accent-soft)] px-4 py-3 text-sm text-[var(--ink)]"
          role="status"
        >
          {notice}
        </div>
      ) : null}
      {error ? (
        <div
          className="rounded-sm border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
          role="alert"
        >
          {error}
        </div>
      ) : null}

      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
            Job sites — {enabledCount} active
          </h3>
          <button
            type="button"
            disabled={checking}
            onClick={() => void testSources()}
            className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-1.5 text-xs disabled:opacity-60"
          >
            {checking ? "Probing sources…" : "Test sources now"}
          </button>
        </div>
        <div className="grid gap-2">
          {sources.map((s) => (
            <SourceRow
              key={s.name}
              source={s}
              checked={settings.sources[s.name] ?? true}
              onToggle={() => toggle(s.name)}
              health={health?.[s.name]}
            />
          ))}
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
          Target regions
        </h3>
        <p className="text-sm text-[var(--muted)]">
          2-letter country codes scope region-aware job sites and the recruiter
          search. A source that covers none of the enabled regions is skipped
          during a search.
        </p>
        <RegionChips
          codes={settings.recruiter_countries}
          onRemove={removeRegion}
        />
        <div className="flex items-center gap-2">
          <input
            value={newRegion}
            onChange={(e) => setNewRegion(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") addRegion(newRegion);
            }}
            placeholder="Add a country code (e.g. de)"
            className="w-60 rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
          />
          <button
            type="button"
            onClick={() => addRegion(newRegion)}
            className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
          >
            Add
          </button>
        </div>
        {data.defaults?.length ? (
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            <span className="text-[var(--muted)]">Suggested:</span>
            {data.defaults
              .filter((c) => !settings.recruiter_countries.includes(c))
              .map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => addRegion(c)}
                  className="mono rounded-sm border border-dashed border-[var(--line)] px-1.5 py-0.5 text-[11px] uppercase text-[var(--muted)] hover:border-[var(--accent)] hover:text-[var(--accent)]"
                >
                  + {c}
                </button>
              ))}
          </div>
        ) : null}
      </section>
    </div>
  );
}

function chipClass(active: boolean) {
  return `mono rounded-sm border px-1.5 py-0.5 text-[10px] uppercase tracking-wide ${
    active
      ? "border-[var(--accent)] bg-[var(--accent-soft)] text-[var(--accent)]"
      : "border-[var(--line)] text-[var(--muted)]"
  }`;
}

function RegionChips({
  codes,
  onRemove,
}: {
  codes: string[];
  onRemove: (code: string) => void;
}) {
  if (codes.length === 0) {
    return (
      <p className="text-xs text-[var(--muted)]">
        No regions enabled — region-aware sources are skipped.
      </p>
    );
  }
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {codes.map((c) => (
        <span
          key={c}
          className="mono inline-flex items-center gap-1 rounded-sm border border-[var(--accent)] bg-[var(--accent-soft)] px-1.5 py-0.5 text-[11px] uppercase text-[var(--accent)]"
        >
          {c}
          <button
            type="button"
            onClick={() => onRemove(c)}
            aria-label={`Remove ${c}`}
            className="hover:text-[var(--ink)]"
          >
            ×
          </button>
        </span>
      ))}
    </div>
  );
}

function SourceRow({
  source,
  checked,
  onToggle,
  health,
}: {
  source: SourceInfo;
  checked: boolean;
  onToggle: () => void;
  health?: SourceHealth;
}) {
  return (
    <label
      className={`flex items-start gap-3 rounded-sm border border-[var(--line)] bg-[var(--panel)] p-3 ${
        source.ready ? "" : "opacity-70"
      }`}
    >
      <input
        type="checkbox"
        checked={checked && source.ready}
        disabled={!source.ready}
        onChange={onToggle}
        className="mt-1 accent-[var(--accent)]"
      />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-sm font-medium text-[var(--ink)]">
            {source.label}
          </span>
          <span className={chipClass(false)}>
            {KIND_LABEL[source.kind] ?? source.kind}
          </span>
          {source.fragile ? (
            <span className="mono rounded-sm border border-amber-300 bg-amber-100 px-1.5 py-0.5 text-[10px] uppercase text-amber-800">
              fragile
            </span>
          ) : null}
          {source.unverified ? (
            <span className={chipClass(false)}>unverified</span>
          ) : null}
          {source.countries.map((c) => (
            <span
              key={c}
              className="mono rounded-sm border border-[var(--line)] px-1 py-0.5 text-[10px] uppercase text-[var(--muted)]"
            >
              {c}
            </span>
          ))}
        </div>
        {source.note ? (
          <p className="mt-1 text-xs text-[var(--muted)]">{source.note}</p>
        ) : null}
        {!source.ready ? (
          <p className="mt-1 text-xs text-amber-700">{source.reason}</p>
        ) : null}
        {health ? (
          <p className="mt-1 text-xs">
            {!health.checked ? (
              <span className="text-[var(--muted)]">
                skipped — {health.reason ?? "off"}
              </span>
            ) : health.ok ? (
              <span className="text-emerald-700">
                live: {health.count} postings (
                {(health.duration_ms / 1000).toFixed(1)}s)
              </span>
            ) : (
              <span className="text-red-700">problem: {health.error}</span>
            )}
          </p>
        ) : null}
      </div>
    </label>
  );
}

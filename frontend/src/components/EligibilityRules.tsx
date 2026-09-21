"use client";

import { useState } from "react";
import {
  api,
  type EligibilityCatalogEntry,
  type EligibilityRules,
  type EligibilityVerdict,
} from "@/lib/api";

type PhraseKey =
  | "worldwide_phrases"
  | "sponsorship_phrases"
  | "relocation_phrases"
  | "country_bounded_phrases"
  | "work_permit_phrases"
  | "work_permit_country_phrases";

const PHRASE_LISTS: Array<{
  key: PhraseKey;
  title: string;
  hint: string;
  placeholder: string;
}> = [
  {
    key: "worldwide_phrases",
    title: "Worldwide / hire-anywhere phrases",
    hint: "A posting containing any of these passes as global remote or hire-anywhere.",
    placeholder: "e.g. work from anywhere",
  },
  {
    key: "sponsorship_phrases",
    title: "Visa sponsorship phrases",
    hint: "Together with a relocation phrase, these mark a role as sponsorship-ready.",
    placeholder: "e.g. we sponsor visas",
  },
  {
    key: "relocation_phrases",
    title: "Relocation phrases",
    hint: "The relocation half of the sponsorship + relocation pass.",
    placeholder: "e.g. relocation package",
  },
  {
    key: "country_bounded_phrases",
    title: "Country-bounded phrases (veto)",
    hint: '{country} stands for any country the posting names, e.g. "remote {country} only" matches "Remote — US only" and "Remote (US)".',
    placeholder: "e.g. must reside in {country}",
  },
  {
    key: "work_permit_phrases",
    title: "Existing work-rights phrases (veto)",
    hint: "Requirements you cannot satisfy without local citizenship or a permit.",
    placeholder: "e.g. must have the right to work",
  },
  {
    key: "work_permit_country_phrases",
    title: "Country-specific work-rights phrases (veto)",
    hint: "The same veto with the country named: {country} is expanded for you.",
    placeholder: "e.g. authorized to work in {country}",
  },
];

const RULE_TOGGLES: Array<{
  key: keyof EligibilityRules;
  label: string;
  verdict: "pass" | "veto";
  hint: string;
}> = [
  {
    key: "worldwide_remote_ok",
    label: "Worldwide / global remote / hire anywhere",
    verdict: "pass",
    hint: "EOR and contractor arrangements count as hire-anywhere.",
  },
  {
    key: "sponsored_relocation_ok",
    label: "On-site or hybrid with visa sponsorship + relocation",
    verdict: "pass",
    hint: "Passes and flags the cover letter to highlight relocation readiness.",
  },
  {
    key: "veto_country_bounded_remote",
    label: "Country-bounded remote (remote — US only, must reside in Poland)",
    verdict: "veto",
    hint: "Rejected before any drafting effort is spent.",
  },
  {
    key: "veto_local_work_permit",
    label: "Requires existing local citizenship or a work permit",
    verdict: "veto",
    hint: "Includes postings that refuse sponsorship outright.",
  },
  {
    key: "honor_work_authorized_countries",
    label: "Exempt countries where I already have work rights",
    verdict: "pass",
    hint: "A country-bounded remote role passes when it is bounded to one of your countries.",
  },
  {
    key: "veto_unknown",
    label: "Reject postings with no eligibility signal",
    verdict: "veto",
    hint: "Off by default: a veto should be evidence, not a guess.",
  },
];

export function EligibilityRulesSection({
  rules,
  defaults,
  catalog,
  onChange,
  onReapply,
  reassessing,
}: {
  rules: EligibilityRules;
  defaults?: EligibilityRules;
  catalog: EligibilityCatalogEntry[];
  onChange: (patch: Partial<EligibilityRules>) => void;
  onReapply: () => void;
  reassessing: boolean;
}) {
  const enabled = rules.enabled ?? true;
  const authorized = rules.work_authorized_countries ?? [];

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
          Eligibility rules
        </h3>
        <button
          type="button"
          disabled={reassessing}
          onClick={onReapply}
          className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-1.5 text-xs disabled:opacity-60"
        >
          {reassessing ? "Re-checking…" : "Re-check existing jobs"}
        </button>
      </div>
      <p className="max-w-2xl text-sm text-[var(--muted)]">
        Applied to every posting before a cover letter is written. Vetoed roles
        are archived with the reason and never cost a drafting request.
      </p>

      <label className="flex items-start gap-3 rounded-sm border border-[var(--accent)] bg-[var(--accent-soft)] p-3">
        <input
          type="checkbox"
          checked={enabled}
          onChange={(e) => onChange({ enabled: e.target.checked })}
          className="mt-1 accent-[var(--accent)]"
        />
        <span>
          <span className="text-sm font-medium text-[var(--ink)]">
            Apply eligibility rules during a search
          </span>
          <span className="mt-1 block text-xs text-[var(--muted)]">
            Off means nothing is filtered: every posting reaches the board.
          </span>
        </span>
      </label>

      <div className={`grid gap-2 ${enabled ? "" : "opacity-60"}`}>
        {RULE_TOGGLES.map((rule) => (
          <label
            key={String(rule.key)}
            className="flex items-start gap-3 rounded-sm border border-[var(--line)] bg-[var(--panel)] p-3"
          >
            <input
              type="checkbox"
              checked={Boolean(rules[rule.key] ?? defaults?.[rule.key] ?? true)}
              onChange={(e) => onChange({ [rule.key]: e.target.checked })}
              className="mt-1 accent-[var(--accent)]"
            />
            <span className="min-w-0 flex-1">
              <span className="flex flex-wrap items-center gap-1.5">
                <span className="text-sm text-[var(--ink)]">{rule.label}</span>
                <span className={verdictChip(rule.verdict)}>{rule.verdict}</span>
              </span>
              <span className="mt-1 block text-xs text-[var(--muted)]">
                {rule.hint}
              </span>
            </span>
          </label>
        ))}
      </div>

      <CountryPicker
        codes={authorized}
        catalog={catalog}
        onChange={(codes) => onChange({ work_authorized_countries: codes })}
      />

      {enabled
        ? PHRASE_LISTS.map((list) => (
            <PhraseList
              key={list.key}
              title={list.title}
              hint={list.hint}
              placeholder={list.placeholder}
              phrases={rules[list.key] ?? []}
              defaults={defaults?.[list.key] ?? []}
              onChange={(phrases) => onChange({ [list.key]: phrases })}
            />
          ))
        : null}
    </section>
  );
}

export function EligibilityTester({ rules }: { rules: EligibilityRules }) {
  const [title, setTitle] = useState("");
  const [location, setLocation] = useState("");
  const [description, setDescription] = useState("");
  const [verdict, setVerdict] = useState<EligibilityVerdict | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function test() {
    setBusy(true);
    setError(null);
    setVerdict(null);
    try {
      const res = await api.previewEligibility({
        title,
        location,
        description,
        rules,
      });
      setVerdict(res.verdict);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Preview failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="space-y-3">
      <h3 className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
        Try a job posting
      </h3>
      <p className="max-w-2xl text-sm text-[var(--muted)]">
        Paste a posting to see the verdict these rules produce. Nothing is saved
        and no search runs — the editor above is tested as it stands, even before
        you save.
      </p>
      <div className="grid gap-2 md:grid-cols-2">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Job title"
          className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
        />
        <input
          value={location}
          onChange={(e) => setLocation(e.target.value)}
          placeholder="Location, e.g. Remote — US only"
          className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
        />
      </div>
      <textarea
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        rows={5}
        placeholder="Paste the job description…"
        className="w-full rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
      />
      <button
        type="button"
        disabled={busy}
        onClick={() => void test()}
        className="rounded-sm border border-[var(--accent)] bg-[var(--accent-soft)] px-4 py-2 text-sm font-medium text-[var(--accent)] disabled:opacity-60"
      >
        {busy ? "Checking…" : "Test against these rules"}
      </button>

      {error ? (
        <p className="text-sm text-red-700" role="alert">
          {error}
        </p>
      ) : null}
      {verdict ? <VerdictCard verdict={verdict} /> : null}
    </section>
  );
}

export function VerdictCard({ verdict }: { verdict: EligibilityVerdict }) {
  const signals = verdict.signals ?? [];
  return (
    <div
      className={`space-y-1.5 rounded-sm border p-3 text-sm ${verdictClass(verdict.status)}`}
      role="status"
    >
      <p className="flex flex-wrap items-center gap-2">
        <span className="mono text-[11px] uppercase tracking-wide">
          {verdict.status === "veto"
            ? "veto — do not apply"
            : verdict.status === "pass"
              ? "pass"
              : "no signal — passed through"}
        </span>
        {verdict.highlights_relocation ? (
          <span className="mono text-[10px] uppercase">
            relocation highlighted in the draft
          </span>
        ) : null}
      </p>
      <p>{verdict.reason}</p>
      {signals.length ? (
        <p className="mono text-[11px] opacity-80">
          matched: {signals.join(", ")}
        </p>
      ) : null}
    </div>
  );
}


function CountryPicker({
  codes,
  catalog,
  onChange,
}: {
  codes: string[];
  catalog: EligibilityCatalogEntry[];
  onChange: (codes: string[]) => void;
}) {
  const [selected, setSelected] = useState("");
  const byCode = new Map(catalog.map((c) => [c.code, c.name]));

  return (
    <div className="space-y-2 rounded-sm border border-[var(--line)] bg-[var(--panel)] p-3">
      <p className="text-sm text-[var(--ink)]">
        Countries where I already have work rights
      </p>
      <p className="text-xs text-[var(--muted)]">
        Used by the exemption above. Empty means every country-bounded remote role
        is vetoed, including one bounded to your own country.
      </p>
      {codes.length ? (
        <div className="flex flex-wrap items-center gap-1.5">
          {codes.map((code) => (
            <span
              key={code}
              className="mono inline-flex items-center gap-1 rounded-sm border border-[var(--accent)] bg-[var(--accent-soft)] px-1.5 py-0.5 text-[11px] uppercase text-[var(--accent)]"
            >
              {byCode.get(code) ?? code}
              <button
                type="button"
                onClick={() => onChange(codes.filter((c) => c !== code))}
                aria-label={`Remove ${code}`}
                className="hover:text-[var(--ink)]"
              >
                ×
              </button>
            </span>
          ))}
        </div>
      ) : (
        <p className="text-xs text-amber-700">
          No countries listed yet — every country-bounded remote role is vetoed.
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <select
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          className="w-64 rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
        >
          <option value="">Add a country…</option>
          {catalog
            .filter((c) => !codes.includes(c.code))
            .map((c) => (
              <option key={c.code} value={c.code}>
                {c.name} ({c.code})
              </option>
            ))}
        </select>
        <button
          type="button"
          disabled={!selected}
          onClick={() => {
            onChange([...codes, selected]);
            setSelected("");
          }}
          className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm disabled:opacity-60"
        >
          Add
        </button>
      </div>
    </div>
  );
}


function PhraseList({
  title,
  hint,
  placeholder,
  phrases,
  defaults,
  onChange,
}: {
  title: string;
  hint: string;
  placeholder: string;
  phrases: string[];
  defaults: string[];
  onChange: (phrases: string[]) => void;
}) {
  const [draft, setDraft] = useState("");
  const suggestions = defaults.filter((d) => !phrases.includes(d)).slice(0, 8);

  function add(value: string) {
    const v = value.trim().toLowerCase();
    if (!v || phrases.includes(v)) return;
    onChange([...phrases, v]);
    setDraft("");
  }

  return (
    <div className="space-y-2 rounded-sm border border-[var(--line)] bg-[var(--panel)] p-3">
      <p className="text-sm text-[var(--ink)]">{title}</p>
      <p className="text-xs text-[var(--muted)]">{hint}</p>
      {phrases.length ? (
        <div className="flex flex-wrap items-center gap-1.5">
          {phrases.map((p) => (
            <span
              key={p}
              className="mono inline-flex items-center gap-1 rounded-sm border border-[var(--line)] px-1.5 py-0.5 text-[11px] text-[var(--muted)]"
            >
              {p}
              <button
                type="button"
                onClick={() => onChange(phrases.filter((x) => x !== p))}
                aria-label={`Remove ${p}`}
                className="hover:text-[var(--ink)]"
              >
                ×
              </button>
            </span>
          ))}
        </div>
      ) : (
        <p className="text-xs text-amber-700">
          No phrases — this signal cannot match anything.
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") add(draft);
          }}
          placeholder={placeholder}
          className="w-72 rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
        />
        <button
          type="button"
          onClick={() => add(draft)}
          className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm"
        >
          Add
        </button>
        {defaults.length ? (
          <button
            type="button"
            onClick={() => onChange([...defaults])}
            className="rounded-sm border border-dashed border-[var(--line)] px-3 py-2 text-xs text-[var(--muted)]"
          >
            Reset to defaults
          </button>
        ) : null}
      </div>
      {suggestions.length ? (
        <div className="flex flex-wrap items-center gap-1.5 text-xs">
          <span className="text-[var(--muted)]">Suggested:</span>
          {suggestions.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => add(s)}
              className="mono rounded-sm border border-dashed border-[var(--line)] px-1.5 py-0.5 text-[11px] text-[var(--muted)] hover:border-[var(--accent)] hover:text-[var(--accent)]"
            >
              + {s}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function verdictChip(verdict: "pass" | "veto") {
  return `mono rounded-sm border px-1.5 py-0.5 text-[10px] uppercase tracking-wide ${
    verdict === "veto"
      ? "border-red-300 bg-red-100 text-red-800 dark:border-red-500/40 dark:bg-red-500/15 dark:text-red-300"
      : "border-emerald-300 bg-emerald-100 text-emerald-800 dark:border-emerald-500/40 dark:bg-emerald-500/15 dark:text-emerald-300"
  }`;
}

function verdictClass(status: string) {
  switch (status) {
    case "veto":
      return "border-red-300 bg-red-50 text-red-900 dark:border-red-500/40 dark:bg-red-500/10 dark:text-red-200";
    case "pass":
      return "border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-500/40 dark:bg-emerald-500/10 dark:text-emerald-200";
    default:
      return "border-[var(--line)] bg-[var(--panel)] text-[var(--ink)]";
  }
}


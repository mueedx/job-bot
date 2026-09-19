"use client";

import { useState } from "react";

export function CoverLetterEditor({
  initial,
  onSave,
}: {
  initial: string;
  onSave: (value: string) => Promise<void>;
}) {
  const [value, setValue] = useState(initial);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  async function save() {
    setSaving(true);
    setMessage(null);
    try {
      await onSave(value);
      setMessage("Saved");
    } catch (e) {
      setMessage(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <h3 className="mono text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]">
          Cover letter
        </h3>
        <button
          type="button"
          onClick={save}
          disabled={saving}
          className="rounded-sm bg-[var(--accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
        >
          {saving ? "Saving…" : "Save"}
        </button>
      </div>
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        rows={12}
        className="w-full rounded-sm border border-[var(--line)] bg-[var(--panel)] p-3 text-sm leading-relaxed text-[var(--ink)] outline-none focus:border-[var(--accent)]"
        placeholder="Draft cover letter markdown…"
      />
      {message ? (
        <p className="text-xs text-[var(--muted)]" role="status">
          {message}
        </p>
      ) : null}
    </div>
  );
}

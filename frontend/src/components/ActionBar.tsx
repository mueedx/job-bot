"use client";

type Props = {
  busy?: boolean;
  onApprove: () => void;
  onDiscard: () => void;
  onSwitchTrack: () => void;
  trackLabel: string;
};

export function ActionBar({
  busy,
  onApprove,
  onDiscard,
  onSwitchTrack,
  trackLabel,
}: Props) {
  return (
    <div className="flex flex-wrap items-center gap-2 border-t border-[var(--line)] pt-4">
      <button
        type="button"
        disabled={busy}
        onClick={onApprove}
        className="rounded-sm bg-[var(--accent)] px-3 py-2 text-sm text-white disabled:opacity-60"
      >
        Approve & Submit
      </button>
      <button
        type="button"
        disabled={busy}
        onClick={onSwitchTrack}
        className="rounded-sm border border-[var(--line)] bg-[var(--panel)] px-3 py-2 text-sm text-[var(--ink)] disabled:opacity-60"
      >
        Switch Track ({trackLabel})
      </button>
      <button
        type="button"
        disabled={busy}
        onClick={onDiscard}
        className="rounded-sm border border-[var(--line)] px-3 py-2 text-sm text-[var(--muted)] hover:border-red-300 hover:text-red-700 disabled:opacity-60"
      >
        Discard
      </button>
    </div>
  );
}

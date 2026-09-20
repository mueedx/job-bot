/**
 * Per-source notes shown on job cards and on the job page.
 *
 * Deliberately conservative: a source appears here only when the condition has
 * been verified against its live apply flow or its API terms. Sources that are
 * known-free (Greenhouse, Lever, Ashby) and sources nobody has checked are both
 * absent — a missing badge is honest, a wrong one is not.
 */
export type SourceNoteKind = "paid" | "attribution";

export type SourceNote = {
  kind: SourceNoteKind;
  /** Short chip label, e.g. "premium". */
  label: string;
  /** Full explanation, shown on hover and in the README source table. */
  text: string;
};

const SOURCE_NOTES: Record<string, SourceNote> = {
  remoteok: {
    kind: "paid",
    label: "paywall",
    text: "Applying may require a RemoteOK Premium subscription or login",
  },
  // Key must be "web3": that is the value the crypto scraper stores in
  // jobs.source (it reads RemoteOK's tag feed, so it inherits the same gate).
  web3: {
    kind: "paid",
    label: "paywall",
    text: "Mirrors RemoteOK listings — applying may require RemoteOK Premium or a login",
  },
};

/** Note for a job source, or null when there is nothing verified to say. */
export function sourceNote(
  source: string | null | undefined,
): SourceNote | null {
  if (!source) return null;
  return SOURCE_NOTES[source.toLowerCase()] ?? null;
}

/** Convert a UTC RFC3339 timestamp to a human-readable relative string ("3d ago", "just now"). */
export function relTime(iso: string): string {
  const t = new Date(iso).getTime();
  // Guard against empty/invalid input (NaN) and Go's zero time
  // ("0001-01-01T00:00:00Z" → a large negative epoch) so we never render
  // "NaNmo ago" or "24168mo ago".
  if (!iso || Number.isNaN(t) || t < 1_000_000_000_000) return "unknown";
  const diff = Date.now() - t;
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

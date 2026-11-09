// formatDate turns "2026-11-14" into "Saturday, November 14, 2026". Noon UTC
// keeps the day the same in every timezone.
export function formatDate(isoDate: string, style: "long" | "short" = "long") {
  return new Date(`${isoDate}T12:00:00Z`).toLocaleDateString("en-US", {
    weekday: style === "long" ? "long" : undefined,
    month: style === "long" ? "long" : "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

// plural("memory", 3) -> "3 memories"
export function plural(word: string, count: number) {
  const forms: Record<string, string> = { memory: "memories" };
  return `${count} ${count === 1 ? word : (forms[word] ?? `${word}s`)}`;
}

// isFuture reports whether a moment is still ahead.
export function isFuture(iso?: string) {
  return Boolean(iso) && new Date(iso!).getTime() > Date.now();
}

// formatEventWhen gives the date, plus the time in the event's own timezone
// when it has one: "Saturday, November 14, 2026 at 7:00 PM EST".
export function formatEventWhen(event: { event_date?: string; event_time?: string; event_at?: string; timezone?: string }) {
  if (!event.event_date) return "";
  // Keeps "December 5, 2026" together when it wraps on a phone.
  const date = formatDate(event.event_date).replace(/ (\d)/g, "\u00A0$1");
  if (!event.event_time || !event.event_at) return date;
  const time = new Date(event.event_at).toLocaleTimeString("en-US", {
    hour: "numeric",
    minute: "2-digit",
    timeZone: event.timezone || "UTC",
    timeZoneName: "short",
  });
  return `${date} at ${time}`;
}

// timeToGo gives "12 days to go", or a closer measure on the last day. It's
// empty once the moment has passed.
export function timeToGo(iso?: string) {
  if (!iso) return "";
  const ms = new Date(iso).getTime() - Date.now();
  if (ms <= 0) return "";
  const hours = ms / 3600000;
  if (hours < 1) return "Less than an hour to go";
  if (hours < 24) return `${plural("hour", Math.floor(hours))} to go`;
  return `${plural("day", Math.floor(hours / 24))} to go`;
}

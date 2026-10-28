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

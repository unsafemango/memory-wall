"use client";
import { useCallback, useRef, useState } from "react";
import { API_URL } from "@/lib/api";
import type { Memory } from "@/lib/types";
import { useLiveChanges } from "@/lib/useLiveChanges";
import { MemoryCard } from "./MemoryCard";

// LiveWall shows the event's memories and keeps them current: when the
// organizer approves, hides, or deletes one, the page updates by itself.
// revealed means the memories just opened, so they all fade in together.
export function LiveWall({ slug, initial, revealed = false }: { slug: string; initial: Memory[]; revealed?: boolean }) {
  const [memories, setMemories] = useState(initial);
  // Memories that arrived after the page loaded, so they can fade in.
  const [fresh, setFresh] = useState<Set<string>>(() => new Set(revealed ? initial.map((m) => m.id) : []));
  const known = useRef(new Set(initial.map((m) => m.id)));

  const refresh = useCallback(() => {
    fetch(`${API_URL}/api/events/${slug}/memories`, { cache: "no-store" })
      .then(async (res) => {
        if (!res.ok) return;
        const data: { memories: Memory[] } = await res.json();
        const added = data.memories.filter((m) => !known.current.has(m.id)).map((m) => m.id);
        data.memories.forEach((m) => known.current.add(m.id));
        if (added.length) setFresh((prev) => new Set([...prev, ...added]));
        setMemories(data.memories);
      })
      .catch(() => {});
  }, [slug]);

  useLiveChanges(`/api/events/${slug}/stream`, refresh);

  if (memories.length === 0) {
    return <p className="empty-state">No memories yet. Check back soon.</p>;
  }
  return (
    <section className="timeline" aria-label="Memories" aria-live="polite">
      {memories.map((m) => (
        <MemoryCard key={m.id} memory={m} isNew={fresh.has(m.id)} />
      ))}
    </section>
  );
}

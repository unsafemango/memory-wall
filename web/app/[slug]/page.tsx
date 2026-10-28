import type { Metadata } from "next";
import { notFound } from "next/navigation";
import type { EventInfo, Memory } from "@/lib/types";
import { Confetti } from "./Confetti";
import { MemoryCard } from "./MemoryCard";

// Server-side requests can use a different address than the browser does,
// e.g. http://api:8080 inside Docker.
const API_URL = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL;

async function getMemories(slug: string): Promise<{ event: EventInfo; memories: Memory[] } | null> {
  const res = await fetch(`${API_URL}/api/events/${slug}/memories`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`memories request failed: ${res.status}`);
  return res.json();
}

function formatDate(isoDate: string) {
  // Dates come as "2026-11-14"; noon UTC keeps the day the same in every timezone.
  return new Date(`${isoDate}T12:00:00Z`).toLocaleDateString("en-US", {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const { slug } = await params;
  const data = await getMemories(slug).catch(() => null);
  return { title: data ? `${data.event.title} · Memory Wall` : "Memory Wall" };
}

export default async function EventPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const data = await getMemories(slug);
  if (!data) notFound();

  const { event, memories } = data;

  return (
    <main className="wall">
      <section className="hero">
        <Confetti />
        <span className="hero-eyebrow">{event.event_date ? formatDate(event.event_date) : "A very special day"}</span>
        <h1 className="name">{event.title}</h1>
        <p className="hero-sub">A few people who love you wanted to leave you something.</p>
      </section>

      {memories.length === 0 ? (
        <p className="empty-state">No memories yet. Check back soon.</p>
      ) : (
        <section className="timeline" aria-label="Memories">
          {memories.map((m) => (
            <MemoryCard key={m.id} memory={m} />
          ))}
        </section>
      )}

      {event.closing_message && (
        <section className="closing-wrap">
          <div className="message-wrap">
            <div className="tag">For {event.title}</div>
            <p>{event.closing_message}</p>
            <div className="signoff">With love, always</div>
          </div>
        </section>
      )}
    </main>
  );
}

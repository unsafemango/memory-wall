import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { isFuture } from "@/lib/format";
import type { EventInfo, Memory } from "@/lib/types";
import { EventStage } from "./EventStage";

// Server-side requests can use a different address than the browser does,
// e.g. http://api:8080 inside Docker.
const API_URL = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL;

async function getMemories(slug: string): Promise<{ event: EventInfo; memories: Memory[]; locked?: boolean } | null> {
  const res = await fetch(`${API_URL}/api/events/${slug}/memories`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`memories request failed: ${res.status}`);
  return res.json();
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

  return (
    <main className="wall">
      <EventStage event={data.event} memories={data.memories} locked={Boolean(data.locked)} upcoming={isFuture(data.event.event_at)} />
    </main>
  );
}

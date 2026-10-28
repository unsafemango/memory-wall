import { notFound } from "next/navigation";
import type { EventInfo, Memory } from "@/lib/types";

// Server-side requests can use a different address than the browser does,
// e.g. http://api:8080 inside Docker.
const API_URL = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL;

async function getMemories(slug: string): Promise<{ event: EventInfo; memories: Memory[] } | null> {
  const res = await fetch(`${API_URL}/api/events/${slug}/memories`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`memories request failed: ${res.status}`);
  return res.json();
}

export default async function EventPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const data = await getMemories(slug);
  if (!data) notFound();

  return (
    <main className="mx-auto max-w-2xl p-6">
      <h1 className="mb-6 text-3xl font-semibold">{data.event.title}</h1>
      <div className="flex flex-col gap-4">
        {data.memories.map((m) => (
          <div key={m.id} className="rounded border p-4">
            <p className="font-medium">{m.sender_name}</p>
            <p className="whitespace-pre-wrap">{m.message}</p>
          </div>
        ))}
      </div>
    </main>
  );
}

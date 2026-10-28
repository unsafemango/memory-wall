"use client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Organizer, OrganizerEvent } from "@/lib/types";
import { smallButtonClass } from "@/lib/ui";
import { EventForm } from "./EventForm";

export default function DashboardPage() {
  const router = useRouter();
  const [organizer, setOrganizer] = useState<Organizer | null>(null);
  const [events, setEvents] = useState<OrganizerEvent[] | null>(null);
  const [error, setError] = useState("");

  const loadEvents = useCallback(async () => {
    const res = await api("/api/organizer/events");
    if (res.ok) setEvents(await res.json());
    else setError("We couldn't load your events.");
  }, []);

  useEffect(() => {
    api("/api/organizer/me")
      .then(async (res) => {
        if (res.status === 401) {
          router.push("/login");
          return;
        }
        setOrganizer(await res.json());
        await loadEvents();
      })
      .catch(() => setError("We couldn't reach the server. Try again in a minute."));
  }, [router, loadEvents]);

  async function logout() {
    await api("/api/auth/logout", { method: "POST" });
    router.push("/login");
  }

  if (error) return <main className="mx-auto max-w-2xl p-6"><p>{error}</p></main>;
  if (!organizer || !events) return <main className="mx-auto max-w-2xl p-6"><p>Loading…</p></main>;

  return (
    <main className="mx-auto w-full max-w-2xl p-6">
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold">Hi {organizer.display_name}</h1>
        <button onClick={logout} className={smallButtonClass}>Log out</button>
      </div>

      <h2 className="mb-2 text-lg font-semibold">Your events</h2>
      {events.length === 0 ? (
        <p className="mb-8 opacity-70">No events yet. Create your first one below.</p>
      ) : (
        <ul className="mb-8 flex flex-col gap-2">
          {events.map((ev) => (
            <li key={ev.id} className="flex flex-wrap items-center justify-between gap-2 rounded border border-black/20 p-3 dark:border-white/30">
              <div>
                <p className="font-medium">{ev.title}</p>
                <p className="text-sm opacity-70">/{ev.slug}</p>
              </div>
              <div className="flex gap-2">
                <Link className={smallButtonClass} href={`/${ev.slug}`}>View page</Link>
                <Link className={smallButtonClass} href={`/dashboard/events/${ev.id}`}>Manage</Link>
              </div>
            </li>
          ))}
        </ul>
      )}

      <h2 className="mb-2 text-lg font-semibold">Create an event</h2>
      <EventForm onSaved={(_, id) => (id ? router.push(`/dashboard/events/${id}`) : loadEvents())} />
    </main>
  );
}

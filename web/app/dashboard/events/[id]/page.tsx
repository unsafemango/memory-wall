"use client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { api, errorMessage } from "@/lib/api";
import type { ModerationItem, OrganizerEvent } from "@/lib/types";
import { smallButtonClass } from "@/lib/ui";
import { EventForm } from "../../EventForm";
import { Invitations } from "./Invitations";

const STATUSES = ["pending", "approved", "rejected"] as const;

type Loaded =
  | { kind: "ok"; event: OrganizerEvent; items: ModerationItem[] }
  | { kind: "logged-out" }
  | { kind: "error"; message: string };

async function fetchEvent(id: string): Promise<Loaded> {
  const [eventRes, itemsRes] = await Promise.all([
    api(`/api/organizer/events/${id}`),
    api(`/api/organizer/events/${id}/submissions`),
  ]);
  if (eventRes.status === 401) return { kind: "logged-out" };
  if (!eventRes.ok || !itemsRes.ok) {
    return { kind: "error", message: eventRes.status === 404 ? "This event doesn't exist." : "We couldn't load this event." };
  }
  return { kind: "ok", event: await eventRes.json(), items: await itemsRes.json() };
}

export default function ManageEventPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [event, setEvent] = useState<OrganizerEvent | null>(null);
  const [items, setItems] = useState<ModerationItem[]>([]);
  const [filter, setFilter] = useState<(typeof STATUSES)[number]>("pending");
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const load = useCallback(() => {
    fetchEvent(id)
      .then((result) => {
        if (result.kind === "logged-out") router.push("/login");
        else if (result.kind === "error") setError(result.message);
        else {
          setEvent(result.event);
          setItems(result.items);
        }
      })
      .catch(() => setError("We couldn't reach the server. Try again in a minute."));
  }, [id, router]);

  useEffect(load, [load]);

  async function act(itemID: string, action: "approve" | "reject" | "delete") {
    setError("");
    const res =
      action === "delete"
        ? await api(`/api/organizer/submissions/${itemID}`, { method: "DELETE" })
        : await api(`/api/organizer/submissions/${itemID}/${action}`, { method: "POST" });
    if (!res.ok) {
      setError(await errorMessage(res));
      return;
    }
    setConfirmDelete(null);
    if (action === "delete") {
      setItems((prev) => prev.filter((it) => it.id !== itemID));
    } else {
      const status = action === "approve" ? "approved" : "rejected";
      setItems((prev) => prev.map((it) => (it.id === itemID ? { ...it, status } : it)));
    }
  }

  if (error && !event) return <main className="mx-auto max-w-2xl p-6"><p>{error}</p></main>;
  if (!event) return <main className="mx-auto max-w-2xl p-6"><p>Loading…</p></main>;

  const shown = items.filter((it) => it.status === filter);

  return (
    <main className="mx-auto w-full max-w-2xl p-6">
      <Link href="/dashboard" className="text-sm underline">← All events</Link>
      <div className="mt-2 mb-6 flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-semibold">{event.title}</h1>
        <Link className={smallButtonClass} href={`/${event.slug}`}>View page</Link>
      </div>

      <Invitations eventID={event.id} />

      <h2 className="mb-2 text-lg font-semibold">Submissions</h2>
      <div className="mb-3 flex gap-2" role="tablist">
        {STATUSES.map((s) => (
          <button
            key={s}
            role="tab"
            aria-selected={filter === s}
            onClick={() => setFilter(s)}
            className={`${smallButtonClass} ${filter === s ? "bg-black text-white dark:bg-white dark:text-black" : ""}`}
          >
            {s[0].toUpperCase() + s.slice(1)} ({items.filter((it) => it.status === s).length})
          </button>
        ))}
      </div>
      {error && <p className="mb-2 text-red-600">{error}</p>}
      {shown.length === 0 ? (
        <p className="mb-8 opacity-70">Nothing {filter} right now.</p>
      ) : (
        <ul className="mb-8 flex flex-col gap-3">
          {shown.map((it) => (
            <li key={it.id} className="rounded border border-black/20 p-3 dark:border-white/30">
              <p className="font-medium">{it.sender_name}</p>
              {it.type === "photo" && it.media_url && (
                // eslint-disable-next-line @next/next/no-img-element
                <img className="mt-2 max-h-64 rounded" src={it.media_url} alt={`Photo from ${it.sender_name}`} />
              )}
              {it.type === "video" && it.media_url && (
                <video className="mt-2 max-h-64 rounded" src={it.media_url} controls playsInline preload="metadata" />
              )}
              {it.message && <p className="mt-2 whitespace-pre-wrap">{it.message}</p>}
              <div className="mt-3 flex flex-wrap gap-2">
                {it.status !== "approved" && (
                  <button className={smallButtonClass} onClick={() => act(it.id, "approve")}>Approve</button>
                )}
                {it.status !== "rejected" && (
                  <button className={smallButtonClass} onClick={() => act(it.id, "reject")}>Reject</button>
                )}
                {confirmDelete === it.id ? (
                  <>
                    <button className={`${smallButtonClass} text-red-600`} onClick={() => act(it.id, "delete")}>
                      Delete for good
                    </button>
                    <button className={smallButtonClass} onClick={() => setConfirmDelete(null)}>Cancel</button>
                  </>
                ) : (
                  <button className={smallButtonClass} onClick={() => setConfirmDelete(it.id)}>Delete</button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <h2 className="mb-2 text-lg font-semibold">Event settings</h2>
      {saved && <p className="mb-2 text-green-700">Saved.</p>}
      <EventForm
        event={event}
        onSaved={() => {
          setSaved(true);
          load();
        }}
      />
    </main>
  );
}

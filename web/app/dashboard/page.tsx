"use client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { formatDate, plural } from "@/lib/format";
import type { Organizer, OrganizerEvent } from "@/lib/types";
import { AppHeader } from "../components/AppHeader";
import { Toast, useToast } from "../components/Toast";
import { copyText } from "../components/copy";
import { EventForm } from "./EventForm";

function EventCard({ event, onCopied }: { event: OrganizerEvent; onCopied: (ok: boolean) => void }) {
  const pending = event.pending_count ?? 0;
  const approved = event.approved_count ?? 0;

  return (
    <li className="panel event-card">
      <div>
        <h2 className="event-card-title">{event.title}</h2>
        {event.event_date && <p className="field-hint">{formatDate(event.event_date, "short")}</p>}
      </div>
      <div className="badges">
        {event.moderation_enabled &&
          (pending > 0 ? (
            <span className="badge badge-attention">{plural("memory", pending)} to review</span>
          ) : (
            <span className="badge badge-good">All caught up</span>
          ))}
        <span className="badge">{plural("memory", approved)} on the page</span>
      </div>
      <button
        type="button"
        className="event-link"
        onClick={async () => onCopied(await copyText(`${window.location.origin}/${event.slug}`))}
        title="Copy page link"
      >
        /{event.slug} <span aria-hidden="true">· Copy</span>
      </button>
      <div className="row event-card-actions">
        <Link className="btn btn-primary btn-small" href={`/dashboard/events/${event.id}`}>
          {pending > 0 ? "Review and invite" : "Manage"}
        </Link>
        <Link className="btn btn-secondary btn-small" href={`/${event.slug}`} target="_blank">
          View page
        </Link>
      </div>
    </li>
  );
}

export default function DashboardPage() {
  const router = useRouter();
  const [organizer, setOrganizer] = useState<Organizer | null>(null);
  const [events, setEvents] = useState<OrganizerEvent[] | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const toast = useToast();

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

  const firstEvent = events?.length === 0;

  return (
    <div className="app">
      <AppHeader mode="signed-in" />
      <main className="app-main">
        {error ? (
          <div className="panel empty"><p>{error}</p></div>
        ) : !organizer || !events ? (
          <p className="page-sub" aria-busy="true">Loading your events…</p>
        ) : (
          <>
            <div className="page-head">
              <div>
                <h1 className="page-title">Hi {organizer.display_name}</h1>
                <p className="page-sub">
                  {firstEvent ? "Let's set up your first memory wall." : "Here are your memory walls."}
                </p>
              </div>
              {!firstEvent && !creating && (
                <button type="button" className="btn btn-primary" onClick={() => setCreating(true)}>
                  New event
                </button>
              )}
            </div>

            {(creating || firstEvent) && (
              <section className="panel" style={{ marginTop: 24 }}>
                <h2 className="section-title">New event</h2>
                <p className="section-sub">You can change any of this later.</p>
                <EventForm
                  onSaved={(_, id) => (id ? router.push(`/dashboard/events/${id}`) : loadEvents())}
                  onCancel={firstEvent ? undefined : () => setCreating(false)}
                />
              </section>
            )}

            {!firstEvent && (
              <ul className="event-grid">
                {events.map((ev) => (
                  <EventCard
                    key={ev.id}
                    event={ev}
                    onCopied={(ok) => toast.show(ok ? "Page link copied" : "Couldn't copy, select the link instead")}
                  />
                ))}
              </ul>
            )}
          </>
        )}
      </main>
      <Toast message={toast.message} />
    </div>
  );
}

"use client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { api, errorMessage } from "@/lib/api";
import { formatDate } from "@/lib/format";
import type { ModerationItem, OrganizerEvent } from "@/lib/types";
import { useOrigin } from "@/lib/useOrigin";
import { AppHeader } from "../../../components/AppHeader";
import { Toast, useToast } from "../../../components/Toast";
import { copyText } from "../../../components/copy";
import { EventForm } from "../../EventForm";
import { Invitations } from "./Invitations";
import { ShareLink } from "./ShareLink";

type Status = ModerationItem["status"];

const TABS: { status: Status; label: string; empty: string }[] = [
  { status: "pending", label: "To review", empty: "You're all caught up. New memories will show up here." },
  { status: "approved", label: "On the page", empty: "Nothing on the page yet. Approve a memory to add it." },
  { status: "rejected", label: "Hidden", empty: "Nothing hidden. Memories you hide from the page end up here." },
];

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

function SubmissionCard({
  item,
  onAction,
}: {
  item: ModerationItem;
  onAction: (id: string, action: "approve" | "reject" | "delete") => Promise<void>;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [busy, setBusy] = useState(false);

  async function run(action: "approve" | "reject" | "delete") {
    setBusy(true);
    await onAction(item.id, action);
    setBusy(false);
  }

  return (
    <li className="panel submission">
      {item.type === "photo" && item.media_url && (
        // eslint-disable-next-line @next/next/no-img-element
        <img className="submission-media" src={item.media_url} alt={`Photo from ${item.sender_name}`} />
      )}
      {item.type === "video" && item.media_url && (
        <video
          className="submission-media"
          src={item.media_url}
          poster={item.thumbnail_url}
          controls
          playsInline
          preload="metadata"
        />
      )}
      {item.message && <p className="submission-message">{item.message}</p>}
      <p className="submission-meta">
        <span className="sender">{item.sender_name}</span>
        <span className="field-hint">
          {new Date(item.created_at).toLocaleDateString(undefined, { month: "short", day: "numeric" })}
        </span>
      </p>
      <div className="row">
        {confirmDelete ? (
          <>
            <button type="button" className="btn btn-danger btn-small" disabled={busy} onClick={() => run("delete")}>
              Yes, delete it
            </button>
            <button type="button" className="btn btn-ghost btn-small" onClick={() => setConfirmDelete(false)}>
              Keep it
            </button>
          </>
        ) : (
          <>
            {item.status !== "approved" && (
              <button type="button" className="btn btn-primary btn-small" disabled={busy} onClick={() => run("approve")}>
                {item.status === "rejected" ? "Put back on the page" : "Approve"}
              </button>
            )}
            {item.status !== "rejected" && (
              <button type="button" className="btn btn-secondary btn-small" disabled={busy} onClick={() => run("reject")}>
                {item.status === "approved" ? "Hide from page" : "Don't show"}
              </button>
            )}
            <button type="button" className="btn btn-ghost btn-small" onClick={() => setConfirmDelete(true)}>
              Delete
            </button>
          </>
        )}
      </div>
    </li>
  );
}

export default function ManageEventPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const origin = useOrigin();
  const toast = useToast();
  const [event, setEvent] = useState<OrganizerEvent | null>(null);
  const [items, setItems] = useState<ModerationItem[]>([]);
  const [filter, setFilter] = useState<Status>("pending");
  const [error, setError] = useState("");

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
    const res =
      action === "delete"
        ? await api(`/api/organizer/submissions/${itemID}`, { method: "DELETE" })
        : await api(`/api/organizer/submissions/${itemID}/${action}`, { method: "POST" });
    if (!res.ok) {
      toast.show(await errorMessage(res));
      return;
    }
    if (action === "delete") {
      setItems((prev) => prev.filter((it) => it.id !== itemID));
      toast.show("Deleted");
    } else {
      const status: Status = action === "approve" ? "approved" : "rejected";
      setItems((prev) => prev.map((it) => (it.id === itemID ? { ...it, status } : it)));
      toast.show(action === "approve" ? "Approved, it's on the page now" : "Hidden from the page");
    }
  }

  if (error && !event) {
    return (
      <div className="app">
        <AppHeader mode="signed-in" />
        <main className="app-main">
          <div className="panel empty">
            <p className="empty-title">{error}</p>
            <Link className="btn btn-secondary" href="/dashboard" style={{ marginTop: 12 }}>Back to your events</Link>
          </div>
        </main>
      </div>
    );
  }

  const pageURL = event ? `${origin}/${event.slug}` : "";
  const counts = Object.fromEntries(TABS.map((t) => [t.status, items.filter((it) => it.status === t.status).length]));
  const activeTab = TABS.find((t) => t.status === filter)!;
  const shown = items.filter((it) => it.status === filter);

  return (
    <div className="app">
      <AppHeader mode="signed-in" />
      <main className="app-main">
        {!event ? (
          <p className="page-sub" aria-busy="true">Loading…</p>
        ) : (
          <>
            <Link href="/dashboard" className="back-link">← Your events</Link>
            <div className="page-head">
              <div>
                <h1 className="page-title">{event.title}</h1>
                {event.event_date && <p className="page-sub">{formatDate(event.event_date)}</p>}
              </div>
            </div>

            <div className="panel share-bar">
              <span className="field-name">Page link</span>
              <code className="share-url">{pageURL}</code>
              <div className="row">
                <button
                  type="button"
                  className="btn btn-secondary btn-small"
                  onClick={async () => toast.show((await copyText(pageURL)) ? "Page link copied" : "Couldn't copy, select the link instead")}
                >
                  Copy
                </button>
                <Link className="btn btn-secondary btn-small" href={`/${event.slug}`} target="_blank">View page</Link>
              </div>
            </div>

            <div className="manage-grid">
              <section className="manage-main">
                <h2 className="section-title">Memories</h2>
                {!event.moderation_enabled && (
                  <p className="notice" style={{ marginBottom: 16 }}>
                    Review is off, so new memories go on the page straight away. You can still hide or delete
                    them here.
                  </p>
                )}
                <div className="tabs" role="tablist" aria-label="Filter memories">
                  {TABS.map((t) => (
                    <button
                      key={t.status}
                      role="tab"
                      type="button"
                      aria-selected={filter === t.status}
                      className="tab"
                      onClick={() => setFilter(t.status)}
                    >
                      {t.label} ({counts[t.status]})
                    </button>
                  ))}
                </div>
                {shown.length === 0 ? (
                  <div className="panel empty">
                    <p>{items.length === 0 ? "No memories yet. Share your invite link to get started." : activeTab.empty}</p>
                  </div>
                ) : (
                  <ul className="submission-list">
                    {shown.map((it) => (
                      <SubmissionCard key={it.id} item={it} onAction={act} />
                    ))}
                  </ul>
                )}
              </section>

              <aside className="manage-side">
                <ShareLink
                  eventID={event.id}
                  eventTitle={event.title}
                  shareURL={event.share_url ?? ""}
                  enabled={event.share_enabled ?? false}
                  moderationEnabled={event.moderation_enabled}
                  onChange={(shareURL, enabled) =>
                    setEvent((prev) => (prev ? { ...prev, share_url: shareURL, share_enabled: enabled } : prev))
                  }
                  onToast={toast.show}
                />
                <Invitations
                  eventID={event.id}
                  eventTitle={event.title}
                  eventSlug={event.slug}
                  eventDate={event.event_date}
                  onToast={toast.show}
                />
                <details className="panel settings">
                  <summary className="section-title">Event settings</summary>
                  <div style={{ marginTop: 16 }}>
                    <EventForm
                      key={JSON.stringify(event)}
                      event={event}
                      onSaved={() => {
                        toast.show("Settings saved");
                        load();
                      }}
                    />
                  </div>
                </details>
              </aside>
            </div>
          </>
        )}
      </main>
      <Toast message={toast.message} />
    </div>
  );
}

"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useParams, usePathname, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { formatDate } from "@/lib/format";
import type { ModerationItem, OrganizerEvent } from "@/lib/types";
import { useLiveChanges } from "@/lib/useLiveChanges";
import { useOrigin } from "@/lib/useOrigin";
import { AppHeader } from "../../../components/AppHeader";
import { Toast, useToast } from "../../../components/Toast";
import { copyText } from "../../../components/copy";
import { ManageProvider } from "./context";

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

export default function ManageLayout({ children }: { children: React.ReactNode }) {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const pathname = usePathname();
  const origin = useOrigin();
  const toast = useToast();
  const showToast = toast.show;
  const [event, setEvent] = useState<OrganizerEvent | null>(null);
  const [items, setItems] = useState<ModerationItem[]>([]);
  const [error, setError] = useState("");

  const reload = useCallback(() => {
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

  useEffect(reload, [reload]);

  // New memories and changes made elsewhere (another tab, another device)
  // show up without a reload, whichever section is open.
  const itemsRef = useRef(items);
  useEffect(() => {
    itemsRef.current = items;
  }, [items]);
  const onLiveChange = useCallback(() => {
    api(`/api/organizer/events/${id}/submissions`)
      .then(async (res) => {
        if (!res.ok) return;
        const fresh: ModerationItem[] = await res.json();
        const known = new Set(itemsRef.current.map((it) => it.id));
        const added = fresh.filter((it) => !known.has(it.id) && it.status === "pending");
        setItems(fresh);
        if (added.length === 1) showToast(`New memory from ${added[0].sender_name}`);
        else if (added.length > 1) showToast(`${added.length} new memories to review`);
      })
      .catch(() => {});
  }, [id, showToast]);
  const live = useLiveChanges(event ? `/api/organizer/events/${id}/stream` : null, onLiveChange);

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

  const base = `/dashboard/events/${id}`;
  const pending = items.filter((it) => it.status === "pending").length;
  const sections = [
    { href: base, label: "Memories", badge: pending },
    { href: `${base}/invite`, label: "Invite people", badge: 0 },
    { href: `${base}/settings`, label: "Event settings", badge: 0 },
  ];

  return (
    <div className="app">
      <AppHeader mode="signed-in" />
      {!event ? (
        <main className="app-main">
          <p className="page-sub" aria-busy="true">Loading…</p>
        </main>
      ) : (
        <div className="manage-shell">
          <aside className="manage-sidebar">
            <div className="sidebar-event">
              <p className="sidebar-title">{event.title}</p>
              {event.event_date && <p className="field-hint">{formatDate(event.event_date, "short")}</p>}
            </div>
            <nav className="sidebar-nav" aria-label="Event sections">
              {sections.map((s) => (
                <Link
                  key={s.href}
                  href={s.href}
                  className={pathname === s.href ? "sidebar-link active" : "sidebar-link"}
                  aria-current={pathname === s.href ? "page" : undefined}
                >
                  <span>{s.label}</span>
                  {s.badge > 0 && <span className="badge badge-attention" aria-label={`${s.badge} to review`}>{s.badge}</span>}
                </Link>
              ))}
            </nav>
            <div className="sidebar-actions">
              <Link className="sidebar-action" href={`/${event.slug}`} target="_blank">View page ↗</Link>
              <button
                type="button"
                className="sidebar-action"
                onClick={async () =>
                  showToast((await copyText(`${origin}/${event.slug}`)) ? "Page link copied" : "Couldn't copy the link")
                }
              >
                Copy page link
              </button>
            </div>
          </aside>
          <main className="manage-content">
            <ManageProvider value={{ event, setEvent, items, setItems, live, reload, toast: showToast }}>
              {children}
            </ManageProvider>
          </main>
        </div>
      )}
      <Toast message={toast.message} />
    </div>
  );
}

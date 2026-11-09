"use client";
import { useState } from "react";
import Link from "next/link";
import { api, errorMessage } from "@/lib/api";
import type { ModerationItem } from "@/lib/types";
import { Gallery } from "../../../[slug]/Gallery";
import { useManage } from "./context";

type Status = ModerationItem["status"];
type Action = "approve" | "reject" | "delete";

const TABS: { status: Status; label: string; empty: string }[] = [
  { status: "pending", label: "To review", empty: "You're all caught up. New memories will show up here." },
  { status: "approved", label: "On the page", empty: "Nothing on the page yet. Approve a memory to add it." },
  { status: "rejected", label: "Hidden", empty: "Nothing hidden. Memories you hide from the page end up here." },
];

function SubmissionCard({ item, onAction }: { item: ModerationItem; onAction: (id: string, action: Action) => Promise<void> }) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [busy, setBusy] = useState(false);

  async function run(action: Action) {
    setBusy(true);
    await onAction(item.id, action);
    setBusy(false);
  }

  return (
    <li className="panel submission">
      {item.files.length > 0 && (
        <div className="submission-media">
          <Gallery files={item.files} sender={item.sender_name} />
        </div>
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

export default function MemoriesPage() {
  const { event, items, setItems, live, toast } = useManage();
  const [filter, setFilter] = useState<Status>("pending");

  async function act(itemID: string, action: Action) {
    const res =
      action === "delete"
        ? await api(`/api/organizer/submissions/${itemID}`, { method: "DELETE" })
        : await api(`/api/organizer/submissions/${itemID}/${action}`, { method: "POST" });
    if (!res.ok) {
      toast(await errorMessage(res));
      return;
    }
    if (action === "delete") {
      setItems((prev) => prev.filter((it) => it.id !== itemID));
      toast("Deleted");
    } else {
      const status: Status = action === "approve" ? "approved" : "rejected";
      setItems((prev) => prev.map((it) => (it.id === itemID ? { ...it, status } : it)));
      toast(action === "approve" ? "Approved, it's on the page now" : "Hidden from the page");
    }
  }

  const activeTab = TABS.find((t) => t.status === filter)!;
  const shown = items.filter((it) => it.status === filter);

  return (
    <>
      <div className="section-head content-head">
        <h1 className="page-title">Memories</h1>
        <span className={live ? "live-dot on" : "live-dot"} title={live ? "New memories show up here automatically" : "Reconnecting…"}>
          {live ? "Live" : "Reconnecting…"}
        </span>
      </div>
      {!event.moderation_enabled && (
        <p className="notice" style={{ marginBottom: 16 }}>
          Review is off, so new memories go on the page straight away. You can still hide or delete them here.
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
            {t.label} ({items.filter((it) => it.status === t.status).length})
          </button>
        ))}
      </div>
      {shown.length === 0 ? (
        <div className="panel empty">
          {items.length === 0 ? (
            <>
              <p className="empty-title">No memories yet</p>
              <p>Share your invite link to get started.</p>
              <Link className="btn btn-primary btn-small" href={`/dashboard/events/${event.id}/invite`} style={{ marginTop: 14 }}>
                Invite people
              </Link>
            </>
          ) : (
            <p>{activeTab.empty}</p>
          )}
        </div>
      ) : (
        <ul className="submission-list">
          {shown.map((it) => (
            <SubmissionCard key={it.id} item={it} onAction={act} />
          ))}
        </ul>
      )}
    </>
  );
}

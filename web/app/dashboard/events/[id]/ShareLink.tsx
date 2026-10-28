"use client";
import { useState } from "react";
import { api, errorMessage } from "@/lib/api";
import { copyText } from "../../../components/copy";
import { inviteMessage } from "./invites";

type Props = {
  eventID: string;
  eventTitle: string;
  shareURL: string;
  enabled: boolean;
  moderationEnabled: boolean;
  onChange: (shareURL: string, enabled: boolean) => void;
  onToast: (message: string) => void;
};

// ShareLink is the event's one reusable invite link, the easiest way to
// invite a whole group.
export function ShareLink({ eventID, eventTitle, shareURL, enabled, moderationEnabled, onChange, onToast }: Props) {
  const [confirmReset, setConfirmReset] = useState(false);
  const [busy, setBusy] = useState(false);

  async function update(path: string, body?: object) {
    setBusy(true);
    try {
      const res = await api(`/api/organizer/events/${eventID}/${path}`, {
        method: "POST",
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!res.ok) {
        onToast(await errorMessage(res));
        return false;
      }
      const data: { share_url: string; share_enabled: boolean } = await res.json();
      onChange(data.share_url, data.share_enabled);
      return true;
    } catch {
      onToast("We couldn't reach the server. Try again in a minute.");
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function toggle() {
    if (await update("share-link", { enabled: !enabled })) {
      onToast(enabled ? "Share link turned off" : "Share link turned on");
    }
  }

  async function reset() {
    if (await update("share-link/reset")) {
      setConfirmReset(false);
      onToast("New link ready. The old one no longer works.");
    }
  }

  async function copy(text: string, label: string) {
    onToast((await copyText(text)) ? `${label} copied` : "Couldn't copy, select the text instead");
  }

  return (
    <section className="panel share-panel">
      <div className="share-panel-head">
        <h2 className="section-title">Invite with one link</h2>
        <label className="switch">
          <input type="checkbox" checked={enabled} disabled={busy} onChange={toggle} />
          <span className="switch-track" aria-hidden="true" />
          <span className="switch-label">{enabled ? "On" : "Off"}</span>
        </label>
      </div>

      {enabled ? (
        <>
          <p className="section-sub">
            Anyone with this link can leave a memory. Post it in a group chat, an email, or anywhere your guests are.
          </p>
          <code className="invite-link share-link">{shareURL}</code>
          <div className="row">
            <button
              type="button"
              className="btn btn-primary btn-small"
              onClick={() => copy(inviteMessage(eventTitle, shareURL), "Invite message")}
            >
              Copy invite message
            </button>
            <button type="button" className="btn btn-secondary btn-small" onClick={() => copy(shareURL, "Link")}>
              Copy link
            </button>
          </div>
          {!moderationEnabled && (
            <p className="notice share-warning">
              Review is off, so anything posted with this link goes straight on the page. Turn review on in Event
              settings if you&apos;re sharing it widely.
            </p>
          )}
        </>
      ) : (
        <p className="section-sub">
          This link is off. Anyone who opens it is told it&apos;s been turned off. Personal links below still work.
        </p>
      )}

      <div className="share-reset">
        {confirmReset ? (
          <>
            <p className="field-hint">
              The current link will stop working right away, and anyone you sent it to will need the new one.
            </p>
            <div className="row">
              <button type="button" className="btn btn-danger btn-small" disabled={busy} onClick={reset}>
                Get a new link
              </button>
              <button type="button" className="btn btn-ghost btn-small" onClick={() => setConfirmReset(false)}>
                Cancel
              </button>
            </div>
          </>
        ) : (
          <button type="button" className="btn btn-ghost btn-small" onClick={() => setConfirmReset(true)}>
            Link shared somewhere it shouldn&apos;t be? Get a new one
          </button>
        )}
      </div>
    </section>
  );
}

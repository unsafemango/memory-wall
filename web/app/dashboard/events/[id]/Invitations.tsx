"use client";
import { useCallback, useEffect, useState } from "react";
import { api, errorMessage } from "@/lib/api";
import type { InvitationInfo } from "@/lib/types";
import { copyText } from "../../../components/copy";

function status(inv: InvitationInfo): { text: string; className: string } {
  if (inv.used_at) return { text: "Used", className: "badge badge-good" };
  if (inv.expires_at && new Date(inv.expires_at) < new Date()) return { text: "Expired", className: "badge" };
  if (inv.expires_at) {
    return { text: `Open until ${new Date(inv.expires_at).toLocaleDateString(undefined, { month: "short", day: "numeric" })}`, className: "badge" };
  }
  return { text: "Open", className: "badge" };
}

type Props = {
  eventID: string;
  eventTitle: string;
  onToast: (message: string) => void;
};

export function Invitations({ eventID, eventTitle, onToast }: Props) {
  const [invitations, setInvitations] = useState<InvitationInfo[]>([]);
  const [recipientName, setRecipientName] = useState("");
  const [expiresInDays, setExpiresInDays] = useState(30);
  const [created, setCreated] = useState<{ link: string; name: string } | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api(`/api/organizer/events/${eventID}/invitations`)
      .then(async (res) => {
        if (res.ok) setInvitations(await res.json());
      })
      .catch(() => setError("We couldn't load invitations."));
  }, [eventID]);

  useEffect(load, [load]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setCreating(true);
    try {
      const res = await api(`/api/organizer/events/${eventID}/invitations`, {
        method: "POST",
        body: JSON.stringify({ recipient_name: recipientName, expires_in_days: expiresInDays }),
      });
      if (!res.ok) {
        setError(await errorMessage(res));
        return;
      }
      const data = await res.json();
      setCreated({ link: data.invite_url, name: recipientName.trim() });
      setRecipientName("");
      load();
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setCreating(false);
    }
  }

  const message = created
    ? `Hi${created.name ? ` ${created.name}` : ""}! I'm collecting memories for ${eventTitle}. ` +
      `Could you leave a note, photo, or video? It only takes a minute: ${created.link}`
    : "";

  async function copy(text: string, label: string) {
    onToast((await copyText(text)) ? `${label} copied` : "Couldn't copy, select the text instead");
  }

  return (
    <section className="panel">
      <h2 className="section-title">Invite people</h2>
      <p className="section-sub">Make a link for each person and send it to them. Each link works once.</p>

      <form onSubmit={handleSubmit} className="form">
        <label className="field">
          <span className="field-name">Who is it for?</span>
          <input
            className="input"
            value={recipientName}
            onChange={(e) => setRecipientName(e.target.value)}
            placeholder="Their name (optional)"
            maxLength={100}
          />
        </label>
        <label className="field">
          <span className="field-name">Link stays open for</span>
          <select className="input" value={expiresInDays} onChange={(e) => setExpiresInDays(Number(e.target.value))}>
            <option value={7}>1 week</option>
            <option value={30}>1 month</option>
            <option value={90}>3 months</option>
          </select>
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button type="submit" disabled={creating} className="btn btn-primary">
          {creating ? "Creating…" : "Create invite link"}
        </button>
      </form>

      {created && (
        <div className="invite-result">
          <p className="field-name">
            {created.name ? `Link for ${created.name} is ready` : "Your link is ready"}
          </p>
          <p className="field-hint">Copy it now. For privacy, it can&apos;t be shown again.</p>
          <code className="invite-link">{created.link}</code>
          <div className="row">
            <button type="button" className="btn btn-primary btn-small" onClick={() => copy(message, "Message")}>
              Copy message
            </button>
            <button type="button" className="btn btn-secondary btn-small" onClick={() => copy(created.link, "Link")}>
              Copy link only
            </button>
          </div>
          <p className="field-hint invite-preview">&ldquo;{message}&rdquo;</p>
        </div>
      )}

      {invitations.length > 0 && (
        <>
          <h3 className="list-title">Sent ({invitations.length})</h3>
          <ul className="invite-list">
            {invitations.map((inv) => {
              const st = status(inv);
              return (
                <li key={inv.id}>
                  <span>{inv.recipient_name || "Unnamed guest"}</span>
                  <span className={st.className}>{st.text}</span>
                </li>
              );
            })}
          </ul>
        </>
      )}
    </section>
  );
}

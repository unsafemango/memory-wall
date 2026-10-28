"use client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { api, errorMessage } from "@/lib/api";
import type { InvitationInfo } from "@/lib/types";
import { copyText } from "../../../components/copy";

const DAY = 24 * 60 * 60 * 1000;

function shortDate(d: Date) {
  return d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
}

function status(inv: InvitationInfo): { text: string; className: string } {
  if (inv.used_at) return { text: "Used", className: "badge badge-good" };
  if (inv.expires_at && new Date(inv.expires_at) < new Date()) return { text: "Expired", className: "badge" };
  if (inv.expires_at) return { text: `Open until ${shortDate(new Date(inv.expires_at))}`, className: "badge" };
  return { text: "Open", className: "badge" };
}

type Duration = { key: string; label: string; days: number };

// The choices for how long a link works. "Until the event" only shows up when
// the event has a date that's still ahead. It runs a day or two past the
// event so late guests can still post; the hint shows the exact end date.
function durations(now: number, eventDate?: string): Duration[] {
  const options: Duration[] = [
    { key: "week", label: "1 week", days: 7 },
    { key: "month", label: "1 month", days: 30 },
    { key: "3months", label: "3 months", days: 90 },
  ];
  if (eventDate) {
    const daysUntil = Math.ceil((new Date(`${eventDate}T23:59:59`).getTime() - now) / DAY) + 1;
    if (daysUntil > 1 && daysUntil <= 365) {
      options.unshift({ key: "event", label: "Until the event", days: daysUntil });
    }
  }
  return options;
}

// Raw invite links are only returned once, so links made in this tab are kept
// in sessionStorage. A reload doesn't lose them; closing the tab does.
function storageKey(eventID: string) {
  return `invite-links:${eventID}`;
}
function readSavedLinks(eventID: string): Record<string, string> {
  try {
    return JSON.parse(sessionStorage.getItem(storageKey(eventID)) || "{}");
  } catch {
    return {};
  }
}
function saveLink(eventID: string, id: string, link: string) {
  try {
    sessionStorage.setItem(storageKey(eventID), JSON.stringify({ ...readSavedLinks(eventID), [id]: link }));
  } catch {
    // Storage can be blocked; the link is still on screen.
  }
}

type Props = {
  eventID: string;
  eventTitle: string;
  eventDate?: string;
  onToast: (message: string) => void;
};

export function Invitations({ eventID, eventTitle, eventDate, onToast }: Props) {
  // The clock is read once, when the panel opens, so the dates don't shift while it's on screen.
  const [now] = useState(() => Date.now());
  const options = useMemo(() => durations(now, eventDate), [now, eventDate]);
  const [invitations, setInvitations] = useState<InvitationInfo[]>([]);
  const [savedLinks, setSavedLinks] = useState<Record<string, string>>({});
  const [recipientName, setRecipientName] = useState("");
  const [durationKey, setDurationKey] = useState(options.find((o) => o.key === "month")!.key);
  const [created, setCreated] = useState<{ link: string; name: string } | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");

  const duration = options.find((o) => o.key === durationKey) ?? options[0];
  const endsOn = new Date(now + duration.days * DAY);

  const load = useCallback(() => {
    api(`/api/organizer/events/${eventID}/invitations`)
      .then(async (res) => {
        if (res.ok) {
          setInvitations(await res.json());
          setSavedLinks(readSavedLinks(eventID));
        }
      })
      .catch(() => setError("We couldn't load invitations."));
  }, [eventID]);

  useEffect(load, [load]);

  function messageFor(link: string, name: string) {
    return (
      `Hi${name ? ` ${name}` : ""}! I'm collecting memories for ${eventTitle}. ` +
      `Could you leave a note, photo, or video? It only takes a minute: ${link}`
    );
  }

  async function copy(text: string, label: string) {
    onToast((await copyText(text)) ? `${label} copied` : "Couldn't copy, select the text instead");
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setCreating(true);
    try {
      const res = await api(`/api/organizer/events/${eventID}/invitations`, {
        method: "POST",
        body: JSON.stringify({ recipient_name: recipientName, expires_in_days: duration.days }),
      });
      if (!res.ok) {
        setError(await errorMessage(res));
        return;
      }
      const data: { id: string; invite_url: string } = await res.json();
      saveLink(eventID, data.id, data.invite_url);
      setCreated({ link: data.invite_url, name: recipientName.trim() });
      setRecipientName("");
      load();
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setCreating(false);
    }
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

        <fieldset className="field choice-field">
          <legend className="field-name">How long should it work?</legend>
          <div className="choices">
            {options.map((o) => (
              <label key={o.key} className="choice">
                <input
                  type="radio"
                  name="duration"
                  value={o.key}
                  checked={durationKey === o.key}
                  onChange={() => setDurationKey(o.key)}
                />
                <span>{o.label}</span>
              </label>
            ))}
          </div>
          <span className="field-hint">Works until {shortDate(endsOn)}.</span>
        </fieldset>

        {error && <p className="form-error" role="alert">{error}</p>}
        <button type="submit" disabled={creating} className="btn btn-primary">
          {creating ? "Creating…" : "Create invite link"}
        </button>
      </form>

      {created && (
        <div className="invite-result" aria-live="polite">
          <div className="invite-result-head">
            <p className="field-name">
              {created.name ? `Link for ${created.name} is ready` : "Your link is ready"}
            </p>
            <button type="button" className="btn btn-ghost btn-small" onClick={() => setCreated(null)} aria-label="Close">
              Done
            </button>
          </div>
          <code className="invite-link">{created.link}</code>
          <div className="row">
            <button
              type="button"
              className="btn btn-primary btn-small"
              onClick={() => copy(messageFor(created.link, created.name), "Message")}
            >
              Copy message
            </button>
            <button type="button" className="btn btn-secondary btn-small" onClick={() => copy(created.link, "Link")}>
              Copy link only
            </button>
          </div>
          <p className="field-hint invite-preview">&ldquo;{messageFor(created.link, created.name)}&rdquo;</p>
        </div>
      )}

      {invitations.length > 0 && (
        <>
          <h3 className="list-title">Sent ({invitations.length})</h3>
          <ul className="invite-list">
            {invitations.map((inv) => {
              const st = status(inv);
              const link = !inv.used_at ? savedLinks[inv.id] : undefined;
              return (
                <li key={inv.id}>
                  <span className="invite-name">{inv.recipient_name || "Unnamed guest"}</span>
                  <span className="invite-row-end">
                    <span className={st.className}>{st.text}</span>
                    {link && (
                      <button
                        type="button"
                        className="btn btn-ghost btn-small"
                        onClick={() => copy(messageFor(link, inv.recipient_name), "Message")}
                        title="Copy the invite message again"
                      >
                        Copy
                      </button>
                    )}
                  </span>
                </li>
              );
            })}
          </ul>
          <p className="field-hint" style={{ marginTop: 10 }}>
            Links you make here can be copied again until you close this tab. After that, make a new one.
          </p>
        </>
      )}
    </section>
  );
}

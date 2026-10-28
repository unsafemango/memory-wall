"use client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { api, errorMessage } from "@/lib/api";
import type { InvitationInfo } from "@/lib/types";
import { copyText } from "../../../components/copy";
import { inviteMessage, readSavedLinks, saveLinks } from "./invites";

const DAY = 24 * 60 * 60 * 1000;
const MAX_NAMES = 200;

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

// parseNames splits pasted text into names: one per line, or separated by
// commas, ignoring blanks and repeats.
function parseNames(text: string) {
  const seen = new Set<string>();
  return text
    .split(/[\n,]/)
    .map((n) => n.trim())
    .filter((n) => {
      if (!n || seen.has(n.toLowerCase())) return false;
      seen.add(n.toLowerCase());
      return true;
    });
}

// csvCell quotes a value for a CSV file.
function csvCell(value: string) {
  return /[",\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value;
}

type CreatedLink = { id: string; name: string; link: string };

type Props = {
  eventID: string;
  eventTitle: string;
  eventSlug: string;
  eventDate?: string;
  onToast: (message: string) => void;
};

export function Invitations({ eventID, eventTitle, eventSlug, eventDate, onToast }: Props) {
  // The clock is read once, when the panel opens, so the dates don't shift while it's on screen.
  const [now] = useState(() => Date.now());
  const options = useMemo(() => durations(now, eventDate), [now, eventDate]);
  const [mode, setMode] = useState<"one" | "list">("one");
  const [invitations, setInvitations] = useState<InvitationInfo[]>([]);
  const [savedLinks, setSavedLinks] = useState<Record<string, string>>({});
  const [recipientName, setRecipientName] = useState("");
  const [namesText, setNamesText] = useState("");
  const [durationKey, setDurationKey] = useState("month");
  const [created, setCreated] = useState<CreatedLink[] | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");

  const duration = options.find((o) => o.key === durationKey) ?? options[0];
  const endsOn = new Date(now + duration.days * DAY);
  const names = useMemo(() => parseNames(namesText), [namesText]);

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

  async function copy(text: string, label: string) {
    onToast((await copyText(text)) ? `${label} copied` : "Couldn't copy, select the text instead");
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (mode === "list" && names.length === 0) {
      setError("Add at least one name, one per line.");
      return;
    }
    if (mode === "list" && names.length > MAX_NAMES) {
      setError(`You can make up to ${MAX_NAMES} links at a time. Split the list and do the rest after.`);
      return;
    }
    setCreating(true);
    try {
      const res =
        mode === "one"
          ? await api(`/api/organizer/events/${eventID}/invitations`, {
              method: "POST",
              body: JSON.stringify({ recipient_name: recipientName, expires_in_days: duration.days }),
            })
          : await api(`/api/organizer/events/${eventID}/invitations/bulk`, {
              method: "POST",
              body: JSON.stringify({ names, expires_in_days: duration.days }),
            });
      if (!res.ok) {
        setError(await errorMessage(res));
        return;
      }
      let links: CreatedLink[];
      if (mode === "one") {
        const data: { id: string; invite_url: string } = await res.json();
        links = [{ id: data.id, name: recipientName.trim(), link: data.invite_url }];
        setRecipientName("");
      } else {
        const data: { id: string; recipient_name: string; invite_url: string }[] = await res.json();
        links = data.map((d) => ({ id: d.id, name: d.recipient_name, link: d.invite_url }));
        setNamesText("");
      }
      saveLinks(eventID, Object.fromEntries(links.map((l) => [l.id, l.link])));
      setCreated(links);
      load();
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setCreating(false);
    }
  }

  function downloadCSV(links: CreatedLink[]) {
    const rows = [
      ["Name", "Link", "Message"],
      ...links.map((l) => [l.name, l.link, inviteMessage(eventTitle, l.link, l.name)]),
    ];
    const csv = rows.map((r) => r.map(csvCell).join(",")).join("\n");
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `${eventSlug}-invite-links.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <section className="panel">
      <h2 className="section-title">Personal links</h2>
      <p className="section-sub">
        For greeting someone by name and seeing who has posted. Each personal link works once.
      </p>

      <div className="choices mode-switch" role="radiogroup" aria-label="Who to invite">
        {(
          [
            ["one", "One person"],
            ["list", "A list of names"],
          ] as const
        ).map(([key, label]) => (
          <label key={key} className="choice">
            <input type="radio" name="invite-mode" checked={mode === key} onChange={() => setMode(key)} />
            <span>{label}</span>
          </label>
        ))}
      </div>

      <form onSubmit={handleSubmit} className="form">
        {mode === "one" ? (
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
        ) : (
          <label className="field">
            <span className="field-name">Names</span>
            <textarea
              className="input"
              value={namesText}
              onChange={(e) => setNamesText(e.target.value)}
              placeholder={"One name per line, or paste a list\nAunt Lena\nMarcus\nPriya"}
              rows={6}
              aria-describedby="names-count"
            />
            <span className="field-hint" id="names-count">
              {names.length === 0
                ? "Paste names from a spreadsheet, a contact list, or type them in."
                : `${names.length} ${names.length === 1 ? "link" : "links"} will be made${names.length > MAX_NAMES ? `, the limit is ${MAX_NAMES}` : ""}.`}
            </span>
          </label>
        )}

        <fieldset className="field choice-field">
          <legend className="field-name">How long should {mode === "one" ? "it" : "they"} work?</legend>
          <div className="choices">
            {options.map((o) => (
              <label key={o.key} className="choice">
                <input
                  type="radio"
                  name="duration"
                  value={o.key}
                  checked={duration.key === o.key}
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
          {creating
            ? "Creating…"
            : mode === "one"
              ? "Create link"
              : names.length > 1
                ? `Create ${names.length} links`
                : "Create links"}
        </button>
      </form>

      {created && created.length === 1 && (
        <div className="invite-result" aria-live="polite">
          <div className="invite-result-head">
            <p className="field-name">
              {created[0].name ? `Link for ${created[0].name} is ready` : "Your link is ready"}
            </p>
            <button type="button" className="btn btn-ghost btn-small" onClick={() => setCreated(null)}>Done</button>
          </div>
          <code className="invite-link">{created[0].link}</code>
          <div className="row">
            <button
              type="button"
              className="btn btn-primary btn-small"
              onClick={() => copy(inviteMessage(eventTitle, created[0].link, created[0].name), "Message")}
            >
              Copy message
            </button>
            <button type="button" className="btn btn-secondary btn-small" onClick={() => copy(created[0].link, "Link")}>
              Copy link only
            </button>
          </div>
          <p className="field-hint invite-preview">&ldquo;{inviteMessage(eventTitle, created[0].link, created[0].name)}&rdquo;</p>
        </div>
      )}

      {created && created.length > 1 && (
        <div className="invite-result" aria-live="polite">
          <div className="invite-result-head">
            <p className="field-name">{created.length} links are ready</p>
            <button type="button" className="btn btn-ghost btn-small" onClick={() => setCreated(null)}>Done</button>
          </div>
          <p className="field-hint">
            Download them as a spreadsheet to send with your email or messaging app, or copy them all at once.
          </p>
          <div className="row">
            <button type="button" className="btn btn-primary btn-small" onClick={() => downloadCSV(created)}>
              Download spreadsheet (CSV)
            </button>
            <button
              type="button"
              className="btn btn-secondary btn-small"
              onClick={() => copy(created.map((l) => `${l.name}: ${l.link}`).join("\n"), "All links")}
            >
              Copy all
            </button>
          </div>
          <ul className="bulk-list">
            {created.map((l) => (
              <li key={l.id}>
                <span className="invite-name">{l.name}</span>
                <button
                  type="button"
                  className="btn btn-ghost btn-small"
                  onClick={() => copy(inviteMessage(eventTitle, l.link, l.name), `Message for ${l.name}`)}
                >
                  Copy message
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {invitations.length > 0 && (
        <details className="sent-list">
          <summary className="list-title">
            Sent ({invitations.length}) · {invitations.filter((i) => i.used_at).length} posted
          </summary>
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
                        onClick={() => copy(inviteMessage(eventTitle, link, inv.recipient_name), "Message")}
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
        </details>
      )}
    </section>
  );
}

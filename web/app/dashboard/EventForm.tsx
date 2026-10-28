"use client";
import { useState } from "react";
import { api, errorMessage } from "@/lib/api";
import { useOrigin } from "@/lib/useOrigin";
import type { OrganizerEvent } from "@/lib/types";

// toSlug turns a title into a link-friendly slug: "Sam's 30th!" -> "sams-30th".
function toSlug(title: string) {
  return title
    .toLowerCase()
    .replace(/['’]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
}

type Props = {
  // When editing, the event to start from. Leave out to create a new event.
  event?: OrganizerEvent;
  onSaved: (slug: string, id?: string) => void;
  onCancel?: () => void;
};

export function EventForm({ event, onSaved, onCancel }: Props) {
  const [title, setTitle] = useState(event?.title ?? "");
  const [slug, setSlug] = useState(event?.slug ?? "");
  const [slugEdited, setSlugEdited] = useState(Boolean(event));
  const [eventDate, setEventDate] = useState(event?.event_date ?? "");
  const [closingMessage, setClosingMessage] = useState(event?.closing_message ?? "");
  const [moderationEnabled, setModerationEnabled] = useState(event?.moderation_enabled ?? true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const host = useOrigin().replace(/^https?:\/\//, "");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const body = JSON.stringify({
        title,
        slug,
        event_date: eventDate,
        closing_message: closingMessage,
        moderation_enabled: moderationEnabled,
      });
      const res = event
        ? await api(`/api/organizer/events/${event.id}`, { method: "PATCH", body })
        : await api("/api/organizer/events", { method: "POST", body });
      if (!res.ok) {
        setError(await errorMessage(res));
        return;
      }
      const created = event ? undefined : await res.json();
      onSaved(slug, created?.id);
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="form">
      <label className="field">
        <span className="field-name">What are you celebrating?</span>
        <input
          className="input"
          value={title}
          onChange={(e) => {
            setTitle(e.target.value);
            if (!slugEdited) setSlug(toSlug(e.target.value));
          }}
          placeholder="Sarah's 30th Birthday"
          required
          maxLength={200}
        />
      </label>

      <div className="field">
        <label className="field-name" htmlFor="slug">Page link</label>
        <div className="input-group">
          <span className="input-prefix">{host}/</span>
          <input
            id="slug"
            className="input"
            value={slug}
            onChange={(e) => {
              setSlug(e.target.value.toLowerCase());
              setSlugEdited(true);
            }}
            placeholder="sarah-30th"
            required
            maxLength={60}
            aria-describedby="slug-hint"
          />
        </div>
        <span className="field-hint" id="slug-hint">
          Lowercase letters, numbers, and dashes. This is the page you&apos;ll share when it&apos;s ready.
        </span>
      </div>

      <label className="field">
        <span className="field-name">Date (optional)</span>
        <input className="input" type="date" value={eventDate} onChange={(e) => setEventDate(e.target.value)} />
      </label>

      <label className="field">
        <span className="field-name">Closing message (optional)</span>
        <textarea
          className="input"
          value={closingMessage}
          onChange={(e) => setClosingMessage(e.target.value)}
          placeholder="A note from you, shown at the end of the page after everyone's memories."
          maxLength={5000}
        />
      </label>

      <label className="check">
        <input type="checkbox" checked={moderationEnabled} onChange={(e) => setModerationEnabled(e.target.checked)} />
        <span>
          <span className="field-name">Review memories before they appear</span>
          <br />
          <span className="field-hint">
            {moderationEnabled
              ? "New memories wait for you to approve them."
              : "New memories go on the page as soon as they're sent."}
          </span>
        </span>
      </label>

      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="row">
        <button type="submit" disabled={saving} className="btn btn-primary">
          {saving ? "Saving…" : event ? "Save changes" : "Create event"}
        </button>
        {onCancel && (
          <button type="button" className="btn btn-ghost" onClick={onCancel}>Cancel</button>
        )}
      </div>
    </form>
  );
}

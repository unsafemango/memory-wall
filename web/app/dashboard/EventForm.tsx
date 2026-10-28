"use client";
import { useState } from "react";
import { api, errorMessage } from "@/lib/api";
import type { OrganizerEvent } from "@/lib/types";
import { buttonClass, inputClass } from "@/lib/ui";

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
};

export function EventForm({ event, onSaved }: Props) {
  const [title, setTitle] = useState(event?.title ?? "");
  const [slug, setSlug] = useState(event?.slug ?? "");
  const [slugEdited, setSlugEdited] = useState(Boolean(event));
  const [eventDate, setEventDate] = useState(event?.event_date ?? "");
  const [closingMessage, setClosingMessage] = useState(event?.closing_message ?? "");
  const [moderationEnabled, setModerationEnabled] = useState(event?.moderation_enabled ?? true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

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
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <label className="flex flex-col gap-1">
        Title
        <input
          className={inputClass}
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
      <label className="flex flex-col gap-1">
        Link
        <span className="flex items-center gap-1">
          <span className="opacity-70">/</span>
          <input
            className={`${inputClass} flex-1`}
            value={slug}
            onChange={(e) => {
              setSlug(e.target.value);
              setSlugEdited(true);
            }}
            placeholder="sarah-30th"
            required
            maxLength={60}
          />
        </span>
      </label>
      <label className="flex flex-col gap-1">
        Date (optional)
        <input className={inputClass} type="date" value={eventDate} onChange={(e) => setEventDate(e.target.value)} />
      </label>
      <label className="flex flex-col gap-1">
        Closing message (optional)
        <textarea
          className={`${inputClass} min-h-24`}
          value={closingMessage}
          onChange={(e) => setClosingMessage(e.target.value)}
          placeholder="Shown at the end of the page, after everyone's memories."
          maxLength={5000}
        />
      </label>
      <label className="flex items-center gap-2">
        <input type="checkbox" checked={moderationEnabled} onChange={(e) => setModerationEnabled(e.target.checked)} />
        Approve submissions before they appear on the page
      </label>
      {error && <p className="text-red-600">{error}</p>}
      <button type="submit" disabled={saving} className={buttonClass}>
        {event ? "Save changes" : "Create event"}
      </button>
    </form>
  );
}

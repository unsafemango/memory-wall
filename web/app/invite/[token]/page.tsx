"use client";
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { API_URL, apiError, errorMessage } from "@/lib/api";
import { formatDate, isFuture, timeToGo } from "@/lib/format";
import { Confetti } from "../../[slug]/Confetti";
import { occasionFor } from "@/lib/occasions";
import { uploadFile, type UploadedFile } from "@/lib/upload";
import { FilePicker, type PickedFile } from "./FilePicker";

const MAX_MESSAGE = 5000;

type Invitation = {
  id: string;
  recipient_name: string;
  // True for the event's share link, which anyone can use more than once.
  shared: boolean;
  event: { title: string; slug: string; event_date?: string; moderation_enabled: boolean; occasion?: string; event_at?: string; hide_until_event?: boolean };
};

type LinkProblem = { title: string; body: string };

// What to tell a guest whose link doesn't work, by API error code.
function linkProblem(code: string): LinkProblem {
  switch (code) {
    case "invitation_used":
      return {
        title: "This link has already been used",
        body: "Each invite link works once. If you'd like to add something else, ask the person who sent it for a new link.",
      };
    case "share_link_off":
      return {
        title: "This link has been turned off",
        body: "The organizer isn't taking new memories through this link right now. Ask them if there's a new one.",
      };
    case "invitation_expired":
      return {
        title: "This link has expired",
        body: "Ask the person who sent it for a new link and you're all set.",
      };
    default:
      return {
        title: "We couldn't find this invitation",
        body: "Check that you copied the whole link, or ask the person who sent it to send it again.",
      };
  }
}

export default function InvitePage() {
  const { token } = useParams<{ token: string }>();
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const [problem, setProblem] = useState<LinkProblem | null>(null);
  const [senderName, setSenderName] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [detail, setDetail] = useState("");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [loadingSuggestions, setLoadingSuggestions] = useState(false);
  const [suggestionError, setSuggestionError] = useState("");
  const [nameError, setNameError] = useState("");
  const [helperOpen, setHelperOpen] = useState(false);
  const [tone, setTone] = useState("");
  // What the message said before a suggestion replaced it, so it can come back.
  const [beforeSuggestion, setBeforeSuggestion] = useState<string | null>(null);
  const [files, setFiles] = useState<PickedFile[]>([]);
  const [uploading, setUploading] = useState<{ number: number; progress: number } | null>(null);
  const sending = useRef(false);
  const messageBox = useRef<HTMLTextAreaElement>(null);
  const nameBox = useRef<HTMLInputElement>(null);

  // The message box grows with what's written, so nothing scrolls out of sight.
  useEffect(() => {
    const box = messageBox.current;
    if (!box) return;
    box.style.height = "auto";
    box.style.height = `${box.scrollHeight}px`;
  }, [message, invitation, submitted]);

  function updateFile(id: string, change: Partial<PickedFile>) {
    setFiles((current) => current.map((it) => (it.id === id ? { ...it, ...change } : it)));
  }

  // Check the link first, so a bad link fails before anyone writes a message.
  useEffect(() => {
    fetch(`${API_URL}/api/invitations/${token}`)
      .then(async (res) => {
        if (res.ok) {
          const inv: Invitation = await res.json();
          setInvitation(inv);
          // The invitation was made for this person, so start with their name.
          setSenderName(inv.recipient_name);
        } else {
          setProblem(linkProblem((await apiError(res)).code));
        }
      })
      .catch(() =>
        setProblem({ title: "We couldn't connect", body: "Check your internet connection and try again in a minute." }),
      );
  }, [token]);

  async function getSuggestions(chosenTone: string) {
    setLoadingSuggestions(true);
    setSuggestionError("");
    try {
      const res = await fetch(`${API_URL}/api/invitations/${token}/suggest-message`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sender_name: senderName, tone: chosenTone, detail }),
      });
      if (!res.ok) {
        setSuggestionError(await errorMessage(res));
        return;
      }
      const data = await res.json();
      setSuggestions(data.suggestions || []);
    } catch {
      setSuggestionError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setLoadingSuggestions(false);
    }
  }

  function pickSuggestion(text: string) {
    setBeforeSuggestion(message);
    setMessage(text);
    setHelperOpen(false);
    messageBox.current?.focus();
  }

  // Share links can be used again, so a guest can leave a second memory.
  function startAnother() {
    setMessage("");
    files.forEach((it) => URL.revokeObjectURL(it.previewURL));
    setFiles([]);
    setBeforeSuggestion(null);
    setHelperOpen(false);
    setSuggestions([]);
    setDetail("");
    setFormError("");
    setSubmitted(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    // Pressing Enter in a field can submit even while the button is disabled.
    if (sending.current) return;
    let missing = false;
    setNameError("");
    setFormError("");
    if (!message.trim() && files.length === 0) {
      setFormError("Write a message or add a photo or video first.");
      messageBox.current?.focus();
      missing = true;
    }
    if (!senderName.trim()) {
      setNameError("Add your name so they know who it's from.");
      nameBox.current?.focus();
      missing = true;
    }
    if (missing) return;

    sending.current = true;
    setSubmitting(true);
    try {
      // One at a time, keeping what's done. If one fails, sending again only
      // uploads the ones that didn't make it.
      const uploaded: UploadedFile[] = [];
      for (const [i, item] of files.entries()) {
        if (item.uploaded) {
          uploaded.push(item.uploaded);
          continue;
        }
        setUploading({ number: i + 1, progress: 0 });
        updateFile(item.id, { progress: 0, failed: false });
        try {
          const done = await uploadFile(token, item.file, (progress) => {
            setUploading({ number: i + 1, progress });
            updateFile(item.id, { progress });
          });
          updateFile(item.id, { progress: null, uploaded: done });
          uploaded.push(done);
        } catch (err) {
          updateFile(item.id, { progress: null, failed: true });
          const reason = err instanceof Error && err.message ? err.message : "The upload failed.";
          setFormError(`${item.file.name} didn't upload. ${reason} Press send to try again, or remove it.`);
          return;
        }
      }

      const res = await fetch(`${API_URL}/api/invitations/${token}/submissions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sender_name: senderName, message, files: uploaded }),
      });
      if (!res.ok) {
        setFormError(await errorMessage(res));
        return;
      }
      setSubmitted(true);
      window.scrollTo({ top: 0 });
    } catch {
      setFormError("We couldn't reach the server. Try again in a minute.");
    } finally {
      sending.current = false;
      setSubmitting(false);
      setUploading(null);
    }
  }

  if (problem) {
    return (
      <main className="wall">
        <section className="invite-card invite-center">
          <h1 className="invite-title">{problem.title}</h1>
          <p className="invite-status">{problem.body}</p>
        </section>
      </main>
    );
  }
  if (!invitation) {
    return (
      <main className="wall" aria-busy="true">
        <section className="invite-card invite-center">
          <p className="invite-status">Opening your invitation…</p>
        </section>
      </main>
    );
  }

  const { event } = invitation;
  const occasion = occasionFor(event.occasion);
  const hiddenForNow = Boolean(event.hide_until_event) && isFuture(event.event_at);
  const toGo = occasion.celebration ? timeToGo(event.event_at) : "";

  if (submitted) {
    return (
      <main className="wall">
        {occasion.effect && <Confetti effect={occasion.effect} burst />}
        <section className="invite-card invite-center">
          <div className="done-badge" aria-hidden="true">♥</div>
          <h1 className="invite-title">Thank you{senderName ? `, ${senderName}` : ""}!</h1>
          <p className="invite-status">
            {hiddenForNow
              ? `Your memory for ${event.title} is saved. It stays a surprise until the event, then everything appears together.`
              : event.moderation_enabled
              ? `Your memory for ${event.title} has been sent. It'll appear on the page once it's been approved.`
              : `Your memory is now on the page for ${event.title}.`}
          </p>
          <div className="hero-actions">
            <Link className="wall-button" href={`/${event.slug}`}>{hiddenForNow ? "See the countdown" : "See the page"}</Link>
            {invitation.shared && (
              <button type="button" className="wall-link link-button" onClick={startAnother}>
                Add another memory
              </button>
            )}
          </div>
        </section>
      </main>
    );
  }

  const activeTone = tone || occasion.tones[0].key;
  const sendLabel = uploading
    ? `Uploading ${files.length > 1 ? `${uploading.number} of ${files.length}` : "your file"}… ${Math.round(uploading.progress * 100)}%`
    : submitting
      ? "Sending…"
      : "Send my memory";

  return (
    <main className="wall">
      <section className="invite-card">
        <p className="invite-eyebrow">
          {invitation.recipient_name ? `Hi ${invitation.recipient_name}, you're invited to add to` : "You're invited to add to"}
        </p>
        <h1 className="invite-title">{event.title}</h1>
        {event.event_date && <p className="invite-date">{formatDate(event.event_date)}</p>}
        {toGo && <p className="invite-togo">{toGo}{hiddenForNow ? ". Your memory stays a surprise until then." : ""}</p>}
        <p className="invite-status invite-intro">
          Write a note, add a photo or video, or both. It only takes a minute.
        </p>

        <form onSubmit={handleSubmit} className="invite-form" noValidate>
          <div className="field">
            <label htmlFor="sender-name" className="field-label">Your name</label>
            <input
              id="sender-name"
              ref={nameBox}
              value={senderName}
              onChange={(e) => {
                setSenderName(e.target.value);
                setNameError("");
              }}
              required
              maxLength={100}
              autoComplete="name"
              placeholder="How it'll appear on the page"
              aria-invalid={Boolean(nameError)}
              aria-describedby={nameError ? "name-error" : undefined}
            />
            {nameError && <p className="form-error" id="name-error" role="alert">{nameError}</p>}
          </div>

          <div className="field">
            <label htmlFor="message" className="field-label">Your message</label>
            <div className="composer">
              <textarea
                id="message"
                ref={messageBox}
                value={message}
                rows={4}
                onChange={(e) => {
                  setMessage(e.target.value);
                  setBeforeSuggestion(null);
                  setFormError("");
                }}
                maxLength={MAX_MESSAGE}
                placeholder={files.length > 0 ? "Add a caption (optional)" : occasion.messagePrompt}
                aria-describedby="message-count"
              />
              <div className="composer-bar">
                <button
                  type="button"
                  className={helperOpen ? "composer-tool on" : "composer-tool"}
                  aria-expanded={helperOpen}
                  aria-controls="writing-helper"
                  onClick={() => setHelperOpen((v) => !v)}
                >
                  <span aria-hidden="true">✦</span> Help me write
                </button>
              </div>
              {helperOpen && (
                <div className="helper" id="writing-helper">
                  <label>
                    What should it mention?
                    <input
                      value={detail}
                      onChange={(e) => setDetail(e.target.value)}
                      placeholder="An inside joke, a trip, a memory"
                      maxLength={300}
                    />
                  </label>
                  <div className="chips" role="radiogroup" aria-label="Tone">
                    {occasion.tones.map((t) => (
                      <button
                        key={t.key}
                        type="button"
                        role="radio"
                        aria-checked={activeTone === t.key}
                        className={activeTone === t.key ? "chip on" : "chip"}
                        onClick={() => setTone(t.key)}
                      >
                        {t.label}
                      </button>
                    ))}
                  </div>
                  <button type="button" className="helper-go" disabled={loadingSuggestions} onClick={() => getSuggestions(activeTone)}>
                    {loadingSuggestions ? "Writing a few ideas…" : suggestions.length > 0 ? "Try again" : "Write 3 ideas"}
                  </button>
                  {suggestionError && <p className="form-error" role="alert">{suggestionError}</p>}
                  <div className="suggestions" aria-live="polite">
                    {suggestions.map((s, i) => (
                      <div className="suggestion" key={i}>
                        <p>{s}</p>
                        <button type="button" className="suggestion-use" onClick={() => pickSuggestion(s)}>Use this</button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
            <span id="message-count" className="char-count">
              {message.length > MAX_MESSAGE - 500 ? `${MAX_MESSAGE - message.length} characters left` : ""}
            </span>
            {beforeSuggestion !== null && (
              <p className="field-hint">
                Added to your message. Make it your own, or{" "}
                <button
                  type="button"
                  className="suggestion-use"
                  onClick={() => {
                    setMessage(beforeSuggestion);
                    setBeforeSuggestion(null);
                  }}
                >
                  undo
                </button>
                .
              </p>
            )}
          </div>

          <FilePicker
            items={files}
            locked={submitting}
            onChange={(next) => {
              setFiles(next);
              setFormError("");
            }}
          />

          {formError && <p className="form-error" role="alert">{formError}</p>}
          <button type="submit" disabled={submitting} className="wall-button">
            {sendLabel}
          </button>
          {event.moderation_enabled && (
            <p className="field-hint invite-footnote">The organizer will see your memory before it goes on the page.</p>
          )}
        </form>
      </section>
    </main>
  );
}

"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { apiError, errorMessage } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { ACCEPTED_FILE_TYPES, uploadFile } from "@/lib/upload";

const API_URL = process.env.NEXT_PUBLIC_API_URL;
const MAX_MESSAGE = 5000;

type Invitation = {
  id: string;
  recipient_name: string;
  event: { title: string; slug: string; event_date?: string; moderation_enabled: boolean };
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

function formatSize(bytes: number) {
  return bytes >= 1 << 20 ? `${(bytes / (1 << 20)).toFixed(1)} MB` : `${Math.max(1, Math.round(bytes / 1024))} KB`;
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
  const [file, setFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const messageBox = useRef<HTMLTextAreaElement>(null);

  // A local preview of the chosen photo, released when the file changes.
  const previewURL = useMemo(() => (file?.type.startsWith("image/") ? URL.createObjectURL(file) : null), [file]);
  useEffect(() => () => {
    if (previewURL) URL.revokeObjectURL(previewURL);
  }, [previewURL]);

  function clearFile() {
    setFile(null);
    if (fileInput.current) fileInput.current.value = "";
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

  async function getSuggestions(tone: string) {
    setLoadingSuggestions(true);
    setSuggestionError("");
    try {
      const res = await fetch(`${API_URL}/api/invitations/${token}/suggest-message`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sender_name: senderName, tone, detail }),
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
    setMessage(text);
    messageBox.current?.focus();
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!message.trim() && !file) {
      setFormError("Write a message or add a photo or video first.");
      messageBox.current?.focus();
      return;
    }
    setSubmitting(true);
    setFormError("");
    try {
      let media = {};
      if (file) {
        setUploadProgress(0);
        media = await uploadFile(token, file, setUploadProgress);
      }
      const res = await fetch(`${API_URL}/api/invitations/${token}/submissions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sender_name: senderName, message, ...media }),
      });
      if (!res.ok) {
        setFormError(await errorMessage(res));
        return;
      }
      setSubmitted(true);
      window.scrollTo({ top: 0 });
    } catch (err) {
      setFormError(err instanceof Error && err.message ? err.message : "We couldn't reach the server. Try again in a minute.");
    } finally {
      setSubmitting(false);
      setUploadProgress(null);
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

  if (submitted) {
    return (
      <main className="wall">
        <section className="invite-card invite-center">
          <div className="done-badge" aria-hidden="true">♥</div>
          <h1 className="invite-title">Thank you{senderName ? `, ${senderName}` : ""}!</h1>
          <p className="invite-status">
            {event.moderation_enabled
              ? `Your memory for ${event.title} has been sent. It'll appear on the page once it's been approved.`
              : `Your memory is now on the page for ${event.title}.`}
          </p>
          <Link className="wall-button" href={`/${event.slug}`}>See the page</Link>
        </section>
      </main>
    );
  }

  const sendLabel =
    uploadProgress !== null
      ? `Uploading… ${Math.round(uploadProgress * 100)}%`
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
        <p className="invite-status invite-intro">
          Write a note, add a photo or video, or both. It only takes a minute.
        </p>

        <form onSubmit={handleSubmit} className="invite-form" noValidate>
          <label>
            Your name
            <input
              value={senderName}
              onChange={(e) => setSenderName(e.target.value)}
              required
              maxLength={100}
              autoComplete="name"
              placeholder="How it'll appear on the page"
            />
          </label>

          <div className="field">
            <label htmlFor="message" className="field-label">Your message</label>
            <textarea
              id="message"
              ref={messageBox}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              maxLength={MAX_MESSAGE}
              placeholder={file ? "Add a caption (optional)" : "Share a memory, a wish, or what they mean to you"}
              aria-describedby="message-count"
            />
            <span id="message-count" className="char-count">
              {message.length > MAX_MESSAGE - 500 ? `${MAX_MESSAGE - message.length} characters left` : ""}
            </span>
          </div>

          <details className="helper" open={suggestions.length > 0 || undefined}>
            <summary className="helper-title">Not sure what to say? Get some ideas</summary>
            <div className="helper-body">
              <input
                value={detail}
                onChange={(e) => setDetail(e.target.value)}
                placeholder="Optional: an inside joke, a trip, a memory…"
                aria-label="Something to mention"
                maxLength={300}
              />
              <div className="chips" role="group" aria-label="Pick a tone">
                {[
                  ["heartfelt", "Heartfelt"],
                  ["funny", "Funny"],
                  ["short", "Short and sweet"],
                ].map(([tone, label]) => (
                  <button key={tone} type="button" className="chip" disabled={loadingSuggestions} onClick={() => getSuggestions(tone)}>
                    {label}
                  </button>
                ))}
              </div>
              {loadingSuggestions && <p className="invite-status" aria-live="polite">Writing a few ideas…</p>}
              {suggestionError && <p className="form-error">{suggestionError}</p>}
              {suggestions.length > 0 && <p className="field-hint">Tap one to use it, then make it your own.</p>}
              {suggestions.map((s, i) => (
                <button type="button" key={i} className="suggestion" onClick={() => pickSuggestion(s)}>
                  {s}
                </button>
              ))}
            </div>
          </details>

          <div className="file-field">
            <span className="field-label" id="file-label">Add a photo or video (optional)</span>
            <label className={file ? "file-picker has-file" : "file-picker"}>
              <input
                ref={fileInput}
                type="file"
                className="visually-hidden"
                accept={ACCEPTED_FILE_TYPES}
                aria-labelledby="file-label"
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
              {previewURL ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img className="file-preview" src={previewURL} alt="" />
              ) : (
                <span className="file-icon" aria-hidden="true">+</span>
              )}
              <span className="file-text">
                <span className="file-name">{file ? file.name : "Choose a photo or video"}</span>
                <span className="file-hint">
                  {file ? formatSize(file.size) : "Photos up to 15 MB, videos up to 100 MB"}
                </span>
              </span>
            </label>
            {file && !submitting && (
              <button type="button" className="file-remove" onClick={clearFile}>
                Remove
              </button>
            )}
          </div>

          {uploadProgress !== null && (
            <progress value={uploadProgress} max={1} aria-label="Upload progress">
              {Math.round(uploadProgress * 100)}%
            </progress>
          )}
          {formError && <p className="form-error" role="alert">{formError}</p>}
          <button type="submit" disabled={submitting || !senderName.trim()} className="wall-button">
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

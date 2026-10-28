"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { ACCEPTED_FILE_TYPES, uploadFile } from "@/lib/upload";

const API_URL = process.env.NEXT_PUBLIC_API_URL;

type Invitation = { id: string; event_id: string; recipient_name: string };

export default function InvitePage() {
  const { token } = useParams<{ token: string }>();
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const [linkError, setLinkError] = useState("");
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

  // Check the link first, so a bad link fails before anyone writes a message.
  useEffect(() => {
    fetch(`${API_URL}/api/invitations/${token}`)
      .then(async (res) => {
        if (res.ok) setInvitation(await res.json());
        else if (res.status === 410) setLinkError(`Sorry, ${await res.text()}. Ask whoever sent it for a new link.`);
        else setLinkError("This invitation link isn't valid. Ask whoever sent it for a new one.");
      })
      .catch(() => setLinkError("We couldn't reach the server. Try again in a minute."));
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
        setSuggestionError(await res.text());
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

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
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
        setFormError(await res.text());
        return;
      }
      setSubmitted(true);
    } catch (err) {
      setFormError(err instanceof Error && err.message ? err.message : "We couldn't reach the server. Try again in a minute.");
    } finally {
      setSubmitting(false);
      setUploadProgress(null);
    }
  }

  if (linkError) {
    return (
      <main className="wall">
        <section className="invite-card">
          <p className="invite-status">{linkError}</p>
        </section>
      </main>
    );
  }
  if (!invitation) {
    return (
      <main className="wall" aria-busy="true">
        <section className="invite-card">
          <p className="invite-status">Loading…</p>
        </section>
      </main>
    );
  }
  if (submitted) {
    return (
      <main className="wall">
        <section className="invite-card">
          <h1 className="invite-title">Thank you, {senderName}!</h1>
          <p className="invite-status">Your memory has been added.</p>
        </section>
      </main>
    );
  }

  return (
    <main className="wall">
      <section className="invite-card">
        <h1 className="invite-title">
          {invitation.recipient_name ? `Hi ${invitation.recipient_name}, leave a memory` : "Leave a memory"}
        </h1>
        <form onSubmit={handleSubmit} className="invite-form">
          <label>
            Your name
            <input value={senderName} onChange={(e) => setSenderName(e.target.value)} required maxLength={100} />
          </label>

          <div className="helper">
            <p className="helper-title">Need help writing something?</p>
            <input
              value={detail}
              onChange={(e) => setDetail(e.target.value)}
              placeholder="Optional: an inside joke, a memory..."
              aria-label="Something to mention"
              maxLength={300}
            />
            <div className="chips">
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
            {loadingSuggestions && <p className="invite-status">Thinking…</p>}
            {suggestionError && <p className="form-error">{suggestionError}</p>}
            {suggestions.map((s, i) => (
              <button type="button" key={i} className="suggestion" onClick={() => setMessage(s)}>
                {s}
              </button>
            ))}
          </div>

          <label>
            Your message
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              required={!file}
              maxLength={5000}
              placeholder={file ? "Optional: add a caption" : undefined}
            />
          </label>
          <label>
            Add a photo or video (optional)
            <input type="file" accept={ACCEPTED_FILE_TYPES} onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          </label>
          {uploadProgress !== null && (
            <progress value={uploadProgress} max={1}>
              {Math.round(uploadProgress * 100)}%
            </progress>
          )}
          {formError && <p className="form-error">{formError}</p>}
          <button type="submit" disabled={submitting} className="wall-button">
            {submitting ? "Sending…" : "Send"}
          </button>
        </form>
      </section>
    </main>
  );
}

"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";

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

  // Check the link first, so a bad link fails before anyone writes a message.
  useEffect(() => {
    fetch(`${API_URL}/api/invitations/${token}`)
      .then(async (res) => {
        if (res.ok) setInvitation(await res.json());
        else setLinkError("This invitation link isn't valid. Ask whoever sent it for a new one.");
      })
      .catch(() => setLinkError("We couldn't reach the server. Try again in a minute."));
  }, [token]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setFormError("");
    try {
      const res = await fetch(`${API_URL}/api/invitations/${token}/submissions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sender_name: senderName, message }),
      });
      if (!res.ok) {
        setFormError(await res.text());
        return;
      }
      setSubmitted(true);
    } catch {
      setFormError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setSubmitting(false);
    }
  }

  if (linkError) {
    return <main className="mx-auto max-w-lg p-6"><p>{linkError}</p></main>;
  }
  if (!invitation) {
    return <main className="mx-auto max-w-lg p-6"><p>Loading…</p></main>;
  }
  if (submitted) {
    return (
      <main className="mx-auto max-w-lg p-6">
        <p>Thank you, {senderName}. Your memory has been added.</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-lg p-6">
      <h1 className="mb-4 text-2xl font-semibold">
        {invitation.recipient_name ? `Hi ${invitation.recipient_name}, leave a memory` : "Leave a memory"}
      </h1>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        <label className="flex flex-col gap-1">
          Your name
          <input
            className="rounded border p-2"
            value={senderName}
            onChange={(e) => setSenderName(e.target.value)}
            required
            maxLength={100}
          />
        </label>
        <label className="flex flex-col gap-1">
          Your message
          <textarea
            className="min-h-32 rounded border p-2"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            required
            maxLength={5000}
          />
        </label>
        {formError && <p className="text-red-600">{formError}</p>}
        <button type="submit" disabled={submitting} className="rounded bg-black px-4 py-2 text-white disabled:opacity-50">
          {submitting ? "Sending…" : "Send"}
        </button>
      </form>
    </main>
  );
}

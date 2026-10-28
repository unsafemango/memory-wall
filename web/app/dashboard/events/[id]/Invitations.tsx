"use client";
import { useCallback, useEffect, useState } from "react";
import { api, errorMessage } from "@/lib/api";
import type { InvitationInfo } from "@/lib/types";
import { buttonClass, inputClass, smallButtonClass } from "@/lib/ui";

function statusText(inv: InvitationInfo) {
  if (inv.used_at) return "Used";
  if (inv.expires_at && new Date(inv.expires_at) < new Date()) return "Expired";
  if (inv.expires_at) return `Expires ${new Date(inv.expires_at).toLocaleDateString()}`;
  return "Open";
}

export function Invitations({ eventID }: { eventID: string }) {
  const [invitations, setInvitations] = useState<InvitationInfo[]>([]);
  const [recipientName, setRecipientName] = useState("");
  const [expiresInDays, setExpiresInDays] = useState(30);
  const [newLink, setNewLink] = useState("");
  const [copied, setCopied] = useState(false);
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
    setCopied(false);
    const res = await api(`/api/organizer/events/${eventID}/invitations`, {
      method: "POST",
      body: JSON.stringify({ recipient_name: recipientName, expires_in_days: expiresInDays }),
    });
    if (!res.ok) {
      setError(await errorMessage(res));
      return;
    }
    const data = await res.json();
    setNewLink(data.invite_url);
    setRecipientName("");
    load();
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(newLink);
      setCopied(true);
    } catch {
      // Clipboard access can be blocked; the link is still selectable below.
    }
  }

  return (
    <section className="mb-8">
      <h2 className="mb-2 text-lg font-semibold">Invite someone</h2>
      <p className="mb-3 text-sm opacity-70">Each link works once, so make one per person and send it to them yourself.</p>
      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-2">
        <label className="flex flex-1 flex-col gap-1">
          Their name
          <input className={inputClass} value={recipientName} onChange={(e) => setRecipientName(e.target.value)} placeholder="Optional" maxLength={100} />
        </label>
        <label className="flex flex-col gap-1">
          Expires in
          <select className={inputClass} value={expiresInDays} onChange={(e) => setExpiresInDays(Number(e.target.value))}>
            <option value={7}>7 days</option>
            <option value={30}>30 days</option>
            <option value={90}>90 days</option>
          </select>
        </label>
        <button type="submit" className={buttonClass}>Create link</button>
      </form>
      {error && <p className="mt-2 text-red-600">{error}</p>}
      {newLink && (
        <div className="mt-3 rounded border border-black/20 p-3 dark:border-white/30">
          <p className="mb-1 text-sm">Copy this link now. It won&apos;t be shown again.</p>
          <div className="flex flex-wrap items-center gap-2">
            <code className="min-w-0 flex-1 break-all select-all">{newLink}</code>
            <button type="button" className={smallButtonClass} onClick={copy}>{copied ? "Copied" : "Copy"}</button>
          </div>
        </div>
      )}
      {invitations.length > 0 && (
        <ul className="mt-4 flex flex-col gap-1 text-sm">
          {invitations.map((inv) => (
            <li key={inv.id} className="flex justify-between gap-2 border-b border-black/10 py-1 dark:border-white/10">
              <span>{inv.recipient_name || "Unnamed invitation"}</span>
              <span className="opacity-70">{statusText(inv)}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

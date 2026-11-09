"use client";
import { useState } from "react";
import { api, errorMessage } from "@/lib/api";

// ConfirmEmailNotice reminds an organizer to confirm their address, and can
// send the link again.
export function ConfirmEmailNotice({ email, toast }: { email: string; toast: (message: string) => void }) {
  const [state, setState] = useState<"idle" | "sending" | "sent">("idle");

  async function resend() {
    setState("sending");
    try {
      const res = await api("/api/organizer/me/send-confirmation", { method: "POST" });
      if (!res.ok) {
        toast(await errorMessage(res));
        setState("idle");
        return;
      }
      setState("sent");
    } catch {
      toast("We couldn't reach the server. Try again in a minute.");
      setState("idle");
    }
  }

  return (
    <div className="notice notice-row">
      <span>
        {state === "sent"
          ? `Sent. Check ${email} for the link.`
          : `Confirm ${email} to get an email when new memories arrive.`}
      </span>
      {state !== "sent" && (
        <button type="button" className="btn btn-secondary btn-small" disabled={state === "sending"} onClick={resend}>
          {state === "sending" ? "Sending…" : "Send the link"}
        </button>
      )}
    </div>
  );
}

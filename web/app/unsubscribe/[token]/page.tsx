"use client";
import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { API_URL, errorMessage } from "@/lib/api";

// This asks for a click instead of acting on load, since email scanners open
// links by themselves and would otherwise turn off people's emails.
export default function UnsubscribePage() {
  const { token } = useParams<{ token: string }>();
  const [state, setState] = useState<"ready" | "working" | "done">("ready");
  const [error, setError] = useState("");

  async function turnOff() {
    setState("working");
    setError("");
    try {
      const res = await fetch(`${API_URL}/api/unsubscribe/${encodeURIComponent(token)}`, { method: "POST" });
      if (!res.ok) {
        setError(await errorMessage(res));
        setState("ready");
        return;
      }
      setState("done");
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
      setState("ready");
    }
  }

  return (
    <main className="wall">
      <section className="invite-card invite-center" aria-live="polite">
        {state === "done" ? (
          <>
            <h1 className="invite-title">Emails are off</h1>
            <p className="invite-status">
              We won&apos;t email you about new memories. Your events and memories are all still there, and you can turn
              these back on in your account settings.
            </p>
            <div className="hero-actions">
              <Link className="wall-button" href="/dashboard/account">Account settings</Link>
            </div>
          </>
        ) : (
          <>
            <h1 className="invite-title">Turn off new memory emails?</h1>
            <p className="invite-status">You can still see every new memory on your dashboard.</p>
            {error && <p className="form-error" role="alert">{error}</p>}
            <div className="hero-actions">
              <button type="button" className="wall-button" disabled={state === "working"} onClick={turnOff}>
                {state === "working" ? "Turning off…" : "Turn off emails"}
              </button>
              <Link className="wall-link" href="/dashboard">Keep them on</Link>
            </div>
          </>
        )}
      </section>
    </main>
  );
}

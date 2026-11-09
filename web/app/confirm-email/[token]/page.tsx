"use client";
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, errorMessage } from "@/lib/api";

export default function ConfirmEmailPage() {
  const { token } = useParams<{ token: string }>();
  const [state, setState] = useState<"working" | "done" | "failed">("working");
  const [message, setMessage] = useState("");
  // A link only works once, so it can't run twice if the effect does.
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    api("/api/auth/confirm-email", { method: "POST", body: JSON.stringify({ token }) })
      .then(async (res) => {
        if (res.ok) {
          setState("done");
          return;
        }
        setMessage(await errorMessage(res));
        setState("failed");
      })
      .catch(() => {
        setMessage("We couldn't reach the server. Try the link again in a minute.");
        setState("failed");
      });
  }, [token]);

  return (
    <main className="wall">
      <section className="invite-card invite-center" aria-live="polite">
        {state === "working" && <h1 className="invite-title">Confirming your email…</h1>}
        {state === "done" && (
          <>
            <div className="done-badge" aria-hidden="true">♥</div>
            <h1 className="invite-title">Your email is confirmed</h1>
            <p className="invite-status">We&apos;ll let you know when new memories arrive.</p>
            <div className="hero-actions">
              <Link className="wall-button" href="/dashboard">Go to your events</Link>
            </div>
          </>
        )}
        {state === "failed" && (
          <>
            <h1 className="invite-title">That link didn&apos;t work</h1>
            <p className="invite-status">{message}. You can send a new one from your account settings.</p>
            <div className="hero-actions">
              <Link className="wall-button" href="/dashboard/account">Account settings</Link>
            </div>
          </>
        )}
      </section>
    </main>
  );
}

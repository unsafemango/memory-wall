"use client";
import { useEffect } from "react";

// Shown if the API is unreachable or returns an error.
export default function EventError({ error, retry }: { error: Error & { digest?: string }; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="wall">
      <section className="hero">
        <h1 className="name">Hmm, that didn&apos;t load</h1>
        <p className="hero-sub">We couldn&apos;t load these memories right now. Give it a moment and try again.</p>
        <button className="wall-button" onClick={() => retry()}>
          Try again
        </button>
      </section>
    </main>
  );
}

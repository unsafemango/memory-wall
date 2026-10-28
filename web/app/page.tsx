import Link from "next/link";
import { Confetti } from "./[slug]/Confetti";

export default function Home() {
  return (
    <main className="wall">
      <section className="hero">
        <Confetti />
        <span className="hero-eyebrow">Birthdays, weddings, send-offs</span>
        <h1 className="name">Memory Wall</h1>
        <p className="hero-sub">
          Send friends a link. They leave a note, photo, or video. You get everything on one page to share.
        </p>
        <div className="hero-actions">
          <Link className="wall-button" href="/signup">Start a wall</Link>
          <Link className="wall-link" href="/login">Log in</Link>
        </div>
      </section>
    </main>
  );
}

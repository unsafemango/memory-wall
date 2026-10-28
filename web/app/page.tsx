import Link from "next/link";
import { Confetti } from "./[slug]/Confetti";

const steps = [
  {
    title: "Create a wall",
    body: "Give it a title and a link, like /sarah-30th. Add a closing message to show at the end.",
  },
  {
    title: "Send each guest a link",
    body: "Every invitation link is personal and works once. Guests don't need an account.",
  },
  {
    title: "Share the finished page",
    body: "Approve what comes in, then send the person you're celebrating one page with everything on it.",
  },
];

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

      <section className="home-preview" aria-label="Example wall">
        <span className="home-preview-label">Example</span>
        <div className="timeline">
          <article className="note-card">
            <p className="msg">Thirty looks good on you. Here&apos;s to more late-night diner runs.</p>
            <span className="sender">Marcus</span>
          </article>
          <article className="polaroid">
            <div className="thumb home-photo" role="img" aria-label="Example photo" />
            <div className="cap">
              <span className="sender">Priya</span>
              <p className="msg">Lisbon, 2024. Same time next year?</p>
            </div>
          </article>
          <article className="note-card">
            <p className="msg">So proud of the person you&apos;ve become. Happy birthday, sweetheart.</p>
            <span className="sender">Aunt Lena</span>
          </article>
        </div>
      </section>

      <section className="home-steps">
        <h2>How it works</h2>
        <ol>
          {steps.map((step, i) => (
            <li key={step.title}>
              <span className="step-number" aria-hidden="true">{i + 1}</span>
              <h3>{step.title}</h3>
              <p>{step.body}</p>
            </li>
          ))}
        </ol>
        <p className="home-extra">
          Stuck on what to write? Guests can ask for three drafts in the tone they want, then edit before sending.
        </p>
      </section>

      <section className="home-cta">
        <h2>Start collecting memories</h2>
        <Link className="wall-button" href="/signup">Create your wall</Link>
      </section>
    </main>
  );
}

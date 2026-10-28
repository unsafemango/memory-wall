// Shown while the server fetches the event's memories.
export default function Loading() {
  return (
    <main className="wall" aria-busy="true">
      <section className="hero">
        <span className="skeleton" style={{ width: 160, height: 26 }} />
        <span className="skeleton" style={{ width: "min(80%, 420px)", height: 56 }} />
        <span className="skeleton" style={{ width: "min(70%, 320px)", height: 22 }} />
      </section>
      <section className="timeline">
        {[0, 1, 2].map((i) => (
          <div key={i} className="note-card">
            <span className="skeleton" style={{ width: "100%", height: 18 }} />
            <span className="skeleton" style={{ width: "70%", height: 18, marginTop: 10 }} />
          </div>
        ))}
      </section>
    </main>
  );
}

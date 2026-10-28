import type { Memory } from "@/lib/types";

// isNew marks a memory that arrived while the page was open, so it fades in.
export function MemoryCard({ memory, isNew = false }: { memory: Memory; isNew?: boolean }) {
  const extra = isNew ? " memory-new" : "";
  if (memory.type === "note" || !memory.media_url) {
    return (
      <article className={`note-card${extra}`}>
        <p className="msg">{memory.message}</p>
        <span className="sender">{memory.sender_name}</span>
      </article>
    );
  }

  return (
    <article className={`polaroid${extra}`}>
      {memory.type === "video" ? (
        <video
          className="thumb"
          src={memory.media_url}
          poster={memory.thumbnail_url}
          controls
          playsInline
          preload="metadata"
        />
      ) : (
        // Signed bucket URLs change on every request, so next/image's optimizer can't cache them.
        // eslint-disable-next-line @next/next/no-img-element
        <img className="thumb" src={memory.media_url} alt={`Photo from ${memory.sender_name}`} loading="lazy" />
      )}
      <div className="cap">
        <span className="sender">{memory.sender_name}</span>
        {memory.message && <p className="msg">{memory.message}</p>}
      </div>
    </article>
  );
}

import type { Memory } from "@/lib/types";

export function MemoryCard({ memory }: { memory: Memory }) {
  if (memory.type === "note" || !memory.media_url) {
    return (
      <article className="note-card">
        <p className="msg">{memory.message}</p>
        <span className="sender">{memory.sender_name}</span>
      </article>
    );
  }

  return (
    <article className="polaroid">
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

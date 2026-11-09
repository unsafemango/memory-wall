import type { Memory } from "@/lib/types";
import { Gallery } from "./Gallery";

// isNew marks a memory that arrived while the page was open, so it fades in.
export function MemoryCard({ memory, isNew = false }: { memory: Memory; isNew?: boolean }) {
  const extra = isNew ? " memory-new" : "";
  if (memory.files.length === 0) {
    return (
      <article className={`note-card${extra}`}>
        <p className="msg">{memory.message}</p>
        <span className="sender">{memory.sender_name}</span>
      </article>
    );
  }

  return (
    <article className={`polaroid${extra}`}>
      <Gallery files={memory.files} sender={memory.sender_name} />
      <div className="cap">
        <span className="sender">{memory.sender_name}</span>
        {memory.message && <p className="msg">{memory.message}</p>}
      </div>
    </article>
  );
}

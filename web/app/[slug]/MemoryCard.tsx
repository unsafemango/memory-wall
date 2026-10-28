import type { Memory } from "@/lib/types";

export function MemoryCard({ memory }: { memory: Memory }) {
  return (
    <div className="rounded border p-4">
      <p className="font-medium">{memory.sender_name}</p>
      {memory.type === "photo" && memory.media_url && (
        // Signed bucket URLs change on every request, so next/image's optimizer can't cache them.
        // eslint-disable-next-line @next/next/no-img-element
        <img className="mt-2 w-full rounded" src={memory.media_url} alt={`Photo from ${memory.sender_name}`} />
      )}
      {memory.type === "video" && memory.media_url && (
        <video className="mt-2 w-full rounded" src={memory.media_url} controls playsInline preload="metadata" />
      )}
      {memory.message && <p className="mt-2 whitespace-pre-wrap">{memory.message}</p>}
    </div>
  );
}

"use client";
import { useEffect, useRef, useState } from "react";
import type { MemoryFile } from "@/lib/types";

const isVideo = (file: MemoryFile) => file.mime_type.startsWith("video/");

function Lightbox({ files, sender, start, onClose }: { files: MemoryFile[]; sender: string; start: number; onClose: () => void }) {
  const [index, setIndex] = useState(start);
  const dialog = useRef<HTMLDialogElement>(null);
  const file = files[index];
  const many = files.length > 1;
  const move = (by: number) => setIndex((i) => (i + by + files.length) % files.length);

  // showModal gives focus trapping and the Esc key without any extra code.
  useEffect(() => {
    dialog.current?.showModal();
  }, []);

  return (
    <dialog
      ref={dialog}
      className="lightbox"
      aria-label={`From ${sender}`}
      onClose={onClose}
      onClick={(e) => {
        if (e.target === dialog.current) dialog.current?.close();
      }}
      onKeyDown={(e) => {
        if (many && e.key === "ArrowRight") move(1);
        if (many && e.key === "ArrowLeft") move(-1);
      }}
    >
      {isVideo(file) ? (
        <video key={file.id} src={file.url} poster={file.thumbnail_url} controls autoPlay playsInline />
      ) : (
        // Signed bucket URLs change on every request, so next/image's optimizer can't cache them.
        // eslint-disable-next-line @next/next/no-img-element
        <img key={file.id} src={file.url} alt={`Photo ${index + 1} from ${sender}`} />
      )}
      <button type="button" className="lightbox-btn lightbox-close" aria-label="Close" onClick={() => dialog.current?.close()}>×</button>
      {many && (
        <>
          <button type="button" className="lightbox-btn lightbox-prev" aria-label="Previous" onClick={() => move(-1)}>‹</button>
          <button type="button" className="lightbox-btn lightbox-next" aria-label="Next" onClick={() => move(1)}>›</button>
          <span className="lightbox-count" aria-live="polite">{index + 1} of {files.length}</span>
        </>
      )}
    </dialog>
  );
}

// Gallery shows a memory's files. One video plays right where it is. Anything
// else is a picture you can tap to see full size.
export function Gallery({ files, sender }: { files: MemoryFile[]; sender: string }) {
  const [open, setOpen] = useState<number | null>(null);

  if (files.length === 1 && isVideo(files[0])) {
    return (
      <video className="thumb" src={files[0].url} poster={files[0].thumbnail_url} controls playsInline preload="metadata" />
    );
  }

  return (
    <>
      <div className={`gallery gallery-${files.length}`}>
        {files.map((file, i) => (
          <button
            type="button"
            key={file.id}
            className="gallery-tile"
            aria-label={`Open ${isVideo(file) ? "video" : "photo"} ${i + 1} of ${files.length} from ${sender}`}
            onClick={() => setOpen(i)}
          >
            {isVideo(file) && !file.thumbnail_url ? (
              <video src={`${file.url}#t=0.1`} muted playsInline preload="metadata" tabIndex={-1} />
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={file.thumbnail_url || file.url} alt="" loading="lazy" />
            )}
            {isVideo(file) && <span className="gallery-play" aria-hidden="true">▶</span>}
          </button>
        ))}
      </div>
      {open !== null && <Lightbox files={files} sender={sender} start={open} onClose={() => setOpen(null)} />}
    </>
  );
}

"use client";
import { useEffect, useRef, useState } from "react";
import { ACCEPTED_FILE_TYPES, MAX_FILES, fileKind, fileProblem, type UploadedFile } from "@/lib/upload";

export type PickedFile = {
  id: string;
  file: File;
  kind: "photo" | "video";
  previewURL: string;
  // Between 0 and 1 while this file is uploading.
  progress: number | null;
  // Set once it's in the bucket, so trying again doesn't upload it twice.
  uploaded: UploadedFile | null;
  failed: boolean;
};

function formatDuration(seconds: number) {
  const whole = Math.round(seconds);
  return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, "0")}`;
}

function Tile({ item, locked, onRemove }: { item: PickedFile; locked: boolean; onRemove: () => void }) {
  const [duration, setDuration] = useState<number | null>(null);
  // Not every browser can show every photo, like an iPhone's HEIC files.
  const [noPreview, setNoPreview] = useState(false);
  const label = `${item.kind === "photo" ? "Photo" : "Video"}: ${item.file.name}`;

  return (
    <li className={item.failed ? "file-tile failed" : "file-tile"}>
      {noPreview ? (
        <span className="file-tile-name" title={item.file.name}>{item.file.name}</span>
      ) : item.kind === "photo" ? (
        // A local preview, which next/image can't optimize.
        // eslint-disable-next-line @next/next/no-img-element
        <img src={item.previewURL} alt={label} onError={() => setNoPreview(true)} />
      ) : (
        <video
          src={`${item.previewURL}#t=0.1`}
          muted
          playsInline
          preload="metadata"
          aria-label={label}
          onLoadedMetadata={(e) => setDuration(e.currentTarget.duration)}
          onError={() => setNoPreview(true)}
        />
      )}
      {item.kind === "video" && (
        <span className="file-tile-duration">
          <span aria-hidden="true">▶</span> {duration !== null && Number.isFinite(duration) ? formatDuration(duration) : "Video"}
        </span>
      )}
      {item.progress !== null && (
        <span className="file-tile-progress" role="progressbar" aria-label={`Uploading ${item.file.name}`} aria-valuenow={Math.round(item.progress * 100)} aria-valuemin={0} aria-valuemax={100}>
          <span style={{ width: `${Math.round(item.progress * 100)}%` }} />
        </span>
      )}
      {item.uploaded && <span className="file-tile-done" aria-label="Uploaded">✓</span>}
      {item.failed && <span className="file-tile-failed">Didn&apos;t upload</span>}
      {!locked && (
        <button type="button" className="file-tile-remove" onClick={onRemove} aria-label={`Remove ${item.file.name}`}>
          ×
        </button>
      )}
    </li>
  );
}

type Props = {
  items: PickedFile[];
  locked: boolean;
  onChange: (items: PickedFile[]) => void;
};

// FilePicker lets a guest choose several photos and videos, one batch at a
// time, and shows each as a thumbnail with its own upload progress.
export function FilePicker({ items, locked, onChange }: Props) {
  const [problem, setProblem] = useState("");
  const nextId = useRef(0);
  const room = MAX_FILES - items.length;

  // Release the local previews when the guest leaves the page.
  const latest = useRef(items);
  useEffect(() => {
    latest.current = items;
  }, [items]);
  useEffect(() => () => latest.current.forEach((it) => URL.revokeObjectURL(it.previewURL)), []);

  function add(e: React.ChangeEvent<HTMLInputElement>) {
    const chosen = Array.from(e.target.files ?? []);
    // Cleared so choosing the same file again after removing it still works.
    e.target.value = "";

    const isAdded = (f: File) =>
      items.some((it) => it.file.name === f.name && it.file.size === f.size && it.file.lastModified === f.lastModified);
    const fresh = chosen.filter((f) => !isAdded(f));
    const problems = fresh.map(fileProblem).filter(Boolean);
    const usable = fresh.filter((f) => !fileProblem(f));
    const fits = usable.slice(0, room);

    if (usable.length > fits.length) {
      problems.push(`A memory can hold ${MAX_FILES} photos and videos, so we added the first ${fits.length === 1 ? "one" : fits.length}.`);
    }
    setProblem(problems.join(" "));
    onChange([
      ...items,
      ...fits.map((file) => ({
        id: `file-${nextId.current++}`,
        file,
        kind: fileKind(file)!,
        previewURL: URL.createObjectURL(file),
        progress: null,
        uploaded: null,
        failed: false,
      })),
    ]);
  }

  function remove(item: PickedFile) {
    URL.revokeObjectURL(item.previewURL);
    setProblem("");
    onChange(items.filter((it) => it !== item));
  }

  const input = (
    <input
      type="file"
      multiple
      className="visually-hidden"
      accept={ACCEPTED_FILE_TYPES}
      disabled={locked}
      onChange={add}
    />
  );

  return (
    <div className="files-field">
      <div className="files-head">
        <span className="field-label" id="files-label">Photos and videos</span>
        {items.length > 0 && <span className="field-hint">{items.length} of {MAX_FILES}</span>}
      </div>

      {items.length === 0 ? (
        <label className="files-empty" aria-labelledby="files-label">
          {input}
          <span className="files-empty-icon" aria-hidden="true">+</span>
          <span className="files-empty-text">Add photos or videos</span>
        </label>
      ) : (
        <ul className="file-tiles" aria-labelledby="files-label">
          {items.map((item) => (
            <Tile key={item.id} item={item} locked={locked} onRemove={() => remove(item)} />
          ))}
          {room > 0 && !locked && (
            <li>
              <label className="file-add">
                {input}
                <span aria-hidden="true" className="file-add-icon">+</span>
                Add more
              </label>
            </li>
          )}
        </ul>
      )}

      {problem && <p className="form-error" role="alert">{problem}</p>}
      <p className="field-hint files-hint">Optional. Photos up to 15 MB, videos up to 100 MB.</p>
    </div>
  );
}

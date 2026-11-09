import { API_URL, errorMessage } from "./api";

export const ACCEPTED_FILE_TYPES = "image/jpeg,image/png,image/webp,image/heic,video/mp4,video/quicktime,.heic,.mov";
export const MAX_FILES = 5;

const MB = 1 << 20;
const TYPES: Record<string, { kind: "photo" | "video"; maxBytes: number }> = {
  "image/jpeg": { kind: "photo", maxBytes: 15 * MB },
  "image/png": { kind: "photo", maxBytes: 15 * MB },
  "image/webp": { kind: "photo", maxBytes: 15 * MB },
  "image/heic": { kind: "photo", maxBytes: 15 * MB },
  "video/mp4": { kind: "video", maxBytes: 100 * MB },
  "video/quicktime": { kind: "video", maxBytes: 100 * MB },
};

// Some browsers leave the type blank for iPhone photos and videos.
function mimeType(file: File) {
  if (file.type) return file.type.toLowerCase();
  const name = file.name.toLowerCase();
  if (name.endsWith(".heic")) return "image/heic";
  if (name.endsWith(".mov")) return "video/quicktime";
  return "";
}

export function fileKind(file: File) {
  return TYPES[mimeType(file)]?.kind ?? null;
}

// fileProblem says why a file can't be added, or returns "" if it can.
export function fileProblem(file: File) {
  const type = TYPES[mimeType(file)];
  if (!type) return `${file.name} isn't a file type we can use. Try a JPEG, PNG, WebP, or HEIC photo, or an MP4 or MOV video.`;
  if (file.size > type.maxBytes) {
    return `${file.name} is too large. ${type.kind === "photo" ? "Photos can be up to 15 MB." : "Videos can be up to 100 MB."}`;
  }
  return "";
}

export type UploadedFile = { storage_key: string; mime_type: string };

// uploadFile asks the API for a presigned URL, then PUTs the file straight to
// the bucket. The file never passes through the API server.
export async function uploadFile(
  token: string,
  file: File,
  onProgress: (fraction: number) => void,
): Promise<UploadedFile> {
  const type = mimeType(file);
  const presignRes = await fetch(`${API_URL}/api/invitations/${token}/uploads`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mime_type: type, size: file.size }),
  });
  if (!presignRes.ok) throw new Error(await errorMessage(presignRes));
  const { upload_url, storage_key } = await presignRes.json();

  // fetch can't report upload progress, XMLHttpRequest can.
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", upload_url);
    xhr.setRequestHeader("Content-Type", type);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    };
    xhr.onload = () => (xhr.status < 300 ? resolve() : reject(new Error("The upload failed.")));
    xhr.onerror = () => reject(new Error("The upload failed."));
    xhr.send(file);
  });

  return { storage_key, mime_type: type };
}

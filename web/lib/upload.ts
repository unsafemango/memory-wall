import { errorMessage } from "./api";

const API_URL = process.env.NEXT_PUBLIC_API_URL;

export const ACCEPTED_FILE_TYPES = "image/jpeg,image/png,image/webp,image/heic,video/mp4,video/quicktime";

// uploadFile asks the API for a presigned URL, then PUTs the file straight to
// the bucket. The file never passes through the API server.
export async function uploadFile(
  token: string,
  file: File,
  onProgress: (fraction: number) => void,
): Promise<{ storage_key: string; mime_type: string }> {
  const presignRes = await fetch(`${API_URL}/api/invitations/${token}/uploads`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mime_type: file.type, size: file.size }),
  });
  if (!presignRes.ok) throw new Error(await errorMessage(presignRes));
  const { upload_url, storage_key } = await presignRes.json();

  // fetch can't report upload progress, XMLHttpRequest can.
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", upload_url);
    xhr.setRequestHeader("Content-Type", file.type);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    };
    xhr.onload = () => (xhr.status < 300 ? resolve() : reject(new Error("The upload failed. Try again.")));
    xhr.onerror = () => reject(new Error("The upload failed. Check your connection and try again."));
    xhr.send(file);
  });

  return { storage_key, mime_type: file.type };
}

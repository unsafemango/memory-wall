const API_URL = process.env.NEXT_PUBLIC_API_URL;

// api calls the Go API from the browser, always sending the session cookie.
export function api(path: string, init: RequestInit = {}) {
  return fetch(`${API_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...init.headers },
  });
}

// errorMessage pulls a readable message out of a failed response.
export async function errorMessage(res: Response): Promise<string> {
  const text = await res.text();
  try {
    const body = JSON.parse(text);
    if (body?.error?.message) return body.error.message;
  } catch {
    // Not JSON, use the plain text.
  }
  return text.trim() || `Something went wrong (${res.status}).`;
}

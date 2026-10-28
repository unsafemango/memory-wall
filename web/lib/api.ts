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
    if (body?.error?.message) return capitalize(body.error.message);
  } catch {
    // Not JSON, use the plain text.
  }
  return capitalize(text.trim()) || `Something went wrong (${res.status}).`;
}

// API messages are lowercase so they read well inside sentences; capitalize
// them when they're shown on their own.
function capitalize(text: string) {
  return text ? text[0].toUpperCase() + text.slice(1) : text;
}

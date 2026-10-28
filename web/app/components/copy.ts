// copyText copies to the clipboard and reports whether it worked. Some
// browsers block clipboard access, so callers should keep the text visible.
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

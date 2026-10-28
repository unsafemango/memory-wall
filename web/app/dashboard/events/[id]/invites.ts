// Helpers shared by the share link and personal invitation panels.

// inviteMessage is the text organizers paste into a chat or email.
export function inviteMessage(eventTitle: string, link: string, name = "") {
  return (
    `Hi${name ? ` ${name}` : ""}! I'm collecting memories for ${eventTitle}. ` +
    `Could you leave a note, photo, or video? It only takes a minute: ${link}`
  );
}

// Personal invite links are only returned once, so links made in this tab are
// kept in sessionStorage. A reload doesn't lose them; closing the tab does.
function storageKey(eventID: string) {
  return `invite-links:${eventID}`;
}

export function readSavedLinks(eventID: string): Record<string, string> {
  try {
    return JSON.parse(sessionStorage.getItem(storageKey(eventID)) || "{}");
  } catch {
    return {};
  }
}

export function saveLinks(eventID: string, links: Record<string, string>) {
  try {
    sessionStorage.setItem(storageKey(eventID), JSON.stringify({ ...readSavedLinks(eventID), ...links }));
  } catch {
    // Storage can be blocked; the links are still on screen.
  }
}

// inviteMessage is the text organizers paste into a chat or email.
export function inviteMessage(eventTitle: string, link: string, name = "") {
  return (
    `Hi${name ? ` ${name}` : ""}! I'm collecting memories for ${eventTitle}. ` +
    `Could you leave a note, photo, or video? It only takes a minute: ${link}`
  );
}

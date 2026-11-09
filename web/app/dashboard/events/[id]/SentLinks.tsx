"use client";
import { useState } from "react";
import type { InvitationInfo } from "@/lib/types";
import { copyText } from "../../../components/copy";
import { inviteMessage } from "./invites";

function shortDate(d: Date) {
  return d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
}

function status(inv: InvitationInfo): { text: string; className: string } {
  if (inv.used_at) return { text: "Posted", className: "badge badge-good" };
  if (inv.expires_at && new Date(inv.expires_at) < new Date()) return { text: "Expired", className: "badge" };
  if (inv.expires_at) return { text: `Open until ${shortDate(new Date(inv.expires_at))}`, className: "badge" };
  return { text: "Open", className: "badge" };
}

type Props = {
  eventTitle: string;
  invitations: InvitationInfo[];
  onToast: (message: string) => void;
};

// SentLinks lists personal links. Open a row to see and copy its link again.
export function SentLinks({ eventTitle, invitations, onToast }: Props) {
  const [openID, setOpenID] = useState<string | null>(null);

  async function copy(text: string, label: string) {
    onToast((await copyText(text)) ? `${label} copied` : "Couldn't copy, select the text instead");
  }

  if (invitations.length === 0) {
    return (
      <section className="panel empty">
        <p className="empty-title">No personal links yet</p>
        <p>Links you make under Personal links will show up here.</p>
      </section>
    );
  }

  const posted = invitations.filter((i) => i.used_at).length;

  return (
    <section className="panel">
      <p className="section-sub">{posted} of {invitations.length} have posted. Select a name to see their link.</p>
      <ul className="invite-list">
        {invitations.map((inv) => {
          const st = status(inv);
          const name = inv.recipient_name || "Unnamed guest";
          const open = openID === inv.id;
          return (
            <li key={inv.id} className={open ? "invite-row open" : "invite-row"}>
              <button
                type="button"
                className="invite-row-toggle"
                aria-expanded={open}
                onClick={() => setOpenID(open ? null : inv.id)}
              >
                <span className="invite-name">{name}</span>
                <span className="invite-row-end">
                  <span className={st.className}>{st.text}</span>
                  <span className="chevron" aria-hidden="true">›</span>
                </span>
              </button>
              {open && (
                <div className="invite-row-detail">
                  {inv.invite_url ? (
                    <>
                      <code className="invite-link">{inv.invite_url}</code>
                      <div className="row">
                        <button
                          type="button"
                          className="btn btn-primary btn-small"
                          onClick={() => copy(inviteMessage(eventTitle, inv.invite_url!, inv.recipient_name), "Message")}
                        >
                          Copy message
                        </button>
                        <button type="button" className="btn btn-secondary btn-small" onClick={() => copy(inv.invite_url!, "Link")}>
                          Copy link
                        </button>
                      </div>
                    </>
                  ) : (
                    <p className="field-hint">
                      {inv.used_at
                        ? `${name} already left a memory with this link.`
                        : st.text === "Expired"
                          ? `This link has expired. Make a new one if ${name} still wants to post.`
                          : "This link was made before links were saved, so it can't be shown again. Make a new one."}
                    </p>
                  )}
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

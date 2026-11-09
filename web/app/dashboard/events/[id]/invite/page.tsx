"use client";
import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { InvitationInfo } from "@/lib/types";
import { useManage } from "../context";
import { Invitations } from "../Invitations";
import { SentLinks } from "../SentLinks";
import { ShareLink } from "../ShareLink";

type Tab = "share" | "personal" | "sent";

export default function InvitePage() {
  const { event, setEvent, toast } = useManage();
  const [tab, setTab] = useState<Tab>("share");
  const [invitations, setInvitations] = useState<InvitationInfo[]>([]);

  const load = useCallback(() => {
    api(`/api/organizer/events/${event.id}/invitations`)
      .then(async (res) => {
        if (res.ok) setInvitations(await res.json());
      })
      .catch(() => toast("We couldn't load your invitations."));
  }, [event.id, toast]);

  useEffect(load, [load]);

  const tabs: { key: Tab; label: string }[] = [
    { key: "share", label: "Share link" },
    { key: "personal", label: "Personal links" },
    { key: "sent", label: `Sent (${invitations.length})` },
  ];

  return (
    <>
      <div className="content-head">
        <h1 className="page-title">Invite people</h1>
        <p className="page-sub">Share one link with everyone, or make personal links.</p>
      </div>
      <div className="tabs" role="tablist" aria-label="Ways to invite">
        {tabs.map((t) => (
          <button
            key={t.key}
            role="tab"
            type="button"
            aria-selected={tab === t.key}
            className="tab"
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>
      {/* Kept mounted so a link you just made is still there when you switch back. */}
      <div hidden={tab !== "share"}>
        <ShareLink
          eventID={event.id}
          eventTitle={event.title}
          shareURL={event.share_url ?? ""}
          enabled={event.share_enabled ?? false}
          moderationEnabled={event.moderation_enabled}
          onChange={(shareURL, enabled) =>
            setEvent((prev) => (prev ? { ...prev, share_url: shareURL, share_enabled: enabled } : prev))
          }
          onToast={toast}
        />
      </div>
      <div hidden={tab !== "personal"}>
        <Invitations
          eventID={event.id}
          eventTitle={event.title}
          eventSlug={event.slug}
          eventDate={event.event_date}
          onCreated={load}
          onToast={toast}
        />
      </div>
      <div hidden={tab !== "sent"}>
        <SentLinks eventTitle={event.title} invitations={invitations} onToast={toast} />
      </div>
    </>
  );
}

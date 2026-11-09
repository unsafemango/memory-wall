export type EventInfo = {
  id: string;
  title: string;
  slug: string;
  event_date?: string;
  closing_message?: string;
  occasion?: string;
  honoree_name?: string;
  // "18:30", when the event has a time.
  event_time?: string;
  // IANA name, like America/Toronto.
  timezone?: string;
  // The exact moment of the event.
  event_at?: string;
  hide_until_event?: boolean;
};

export type MemoryFile = {
  id: string;
  mime_type: string;
  url?: string;
  // Set for videos once the background job has made a thumbnail.
  thumbnail_url?: string;
};

export type Memory = {
  id: string;
  sender_name: string;
  type: "note" | "photo" | "video";
  message: string;
  created_at: string;
  files: MemoryFile[];
};

export type OrganizerEvent = EventInfo & {
  closing_message?: string;
  moderation_enabled: boolean;
  // Only included in the dashboard's event list.
  pending_count?: number;
  approved_count?: number;
  // The event's reusable link. Only the owner's endpoints include it.
  share_url?: string;
  share_enabled?: boolean;
};

export type ModerationItem = Memory & {
  status: "pending" | "approved" | "rejected";
};

export type Organizer = {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
  email_verified: boolean;
  notify_new_memories: boolean;
};

export type InvitationInfo = {
  id: string;
  recipient_name: string;
  expires_at?: string;
  used_at?: string;
  // Present while the link can still be used.
  invite_url?: string;
};

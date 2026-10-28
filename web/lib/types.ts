export type EventInfo = {
  id: string;
  title: string;
  slug: string;
  event_date?: string;
  closing_message?: string;
};

export type Memory = {
  id: string;
  sender_name: string;
  type: "note" | "photo" | "video";
  message: string;
  created_at: string;
  mime_type?: string;
  media_url?: string;
  // Set for videos once the background job has made a thumbnail.
  thumbnail_url?: string;
};

export type OrganizerEvent = EventInfo & {
  closing_message?: string;
  moderation_enabled: boolean;
};

export type ModerationItem = Memory & {
  status: "pending" | "approved" | "rejected";
};

export type Organizer = {
  id: string;
  email: string;
  display_name: string;
};

export type InvitationInfo = {
  id: string;
  recipient_name: string;
  expires_at?: string;
  used_at?: string;
};

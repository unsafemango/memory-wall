export type EventInfo = {
  id: string;
  title: string;
  slug: string;
  event_date?: string;
};

export type Memory = {
  id: string;
  sender_name: string;
  type: "note" | "photo" | "video";
  message: string;
  created_at: string;
  mime_type?: string;
  media_url?: string;
};

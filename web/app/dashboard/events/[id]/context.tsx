"use client";
import { createContext, useContext } from "react";
import type { ModerationItem, OrganizerEvent } from "@/lib/types";

export type ManageContextValue = {
  event: OrganizerEvent;
  setEvent: React.Dispatch<React.SetStateAction<OrganizerEvent | null>>;
  items: ModerationItem[];
  setItems: React.Dispatch<React.SetStateAction<ModerationItem[]>>;
  // Whether the live update stream is connected.
  live: boolean;
  reload: () => void;
  toast: (message: string) => void;
};

const ManageContext = createContext<ManageContextValue | null>(null);

export const ManageProvider = ManageContext.Provider;

// useManage gives a manage section the event and its memories, which the
// layout loads once and keeps current.
export function useManage() {
  const value = useContext(ManageContext);
  if (!value) throw new Error("useManage must be used inside the manage layout");
  return value;
}

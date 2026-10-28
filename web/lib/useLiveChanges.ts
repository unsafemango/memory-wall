"use client";
import { useEffect, useRef, useState } from "react";

const API_URL = process.env.NEXT_PUBLIC_API_URL;

export type LiveChange = {
  op: string;
  submission_id?: string;
  status?: string;
  old_status?: string;
};

// useLiveChanges listens to one of the API's change streams and calls
// onChange whenever something changes. Bursts are grouped into a single call,
// and after a dropped connection comes back onChange runs once more, in case
// a change was missed while it was down. Returns whether the stream is live.
export function useLiveChanges(path: string | null, onChange: (changes: LiveChange[]) => void) {
  const callback = useRef(onChange);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    callback.current = onChange;
  }, [onChange]);

  useEffect(() => {
    if (!path) return;
    // The browser reconnects an EventSource by itself after a drop.
    const source = new EventSource(`${API_URL}${path}`, { withCredentials: true });
    let pending: LiveChange[] = [];
    let timer: ReturnType<typeof setTimeout> | undefined;
    let opened = false;

    const flush = () => {
      const batch = pending;
      pending = [];
      callback.current(batch);
    };
    const queue = (change: LiveChange) => {
      pending.push(change);
      clearTimeout(timer);
      timer = setTimeout(flush, 300);
    };

    source.onopen = () => {
      setConnected(true);
      if (opened) queue({ op: "RESYNC" });
      opened = true;
    };
    source.onerror = () => setConnected(false);
    source.addEventListener("change", (e) => {
      try {
        queue(JSON.parse((e as MessageEvent).data));
      } catch {
        queue({ op: "RESYNC" });
      }
    });

    return () => {
      clearTimeout(timer);
      source.close();
    };
  }, [path]);

  return connected;
}

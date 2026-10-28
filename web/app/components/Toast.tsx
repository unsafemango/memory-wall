"use client";
import { useCallback, useEffect, useState } from "react";

// useToast returns a short confirmation message and a function to show one.
// Render <Toast message={message} /> once on the page.
export function useToast() {
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!message) return;
    const timer = setTimeout(() => setMessage(""), 2600);
    return () => clearTimeout(timer);
  }, [message]);

  const show = useCallback((text: string) => setMessage(text), []);
  return { message, show };
}

export function Toast({ message }: { message: string }) {
  // The live region stays mounted so screen readers announce each message.
  return (
    <div role="status" aria-live="polite">
      {message && <div className="toast">{message}</div>}
    </div>
  );
}

"use client";
import { useEffect, useRef, useState } from "react";

function parts(ms: number) {
  const seconds = Math.max(0, Math.floor(ms / 1000));
  return {
    days: Math.floor(seconds / 86400),
    hours: Math.floor(seconds / 3600) % 24,
    minutes: Math.floor(seconds / 60) % 60,
    seconds: seconds % 60,
  };
}

// Countdown ticks down to target and calls onDone once when it gets there. The
// numbers start blank, since the server and browser clocks wouldn't match.
export function Countdown({ target, onDone }: { target: string; onDone: () => void }) {
  const [left, setLeft] = useState<number | null>(null);
  const done = useRef(onDone);

  useEffect(() => {
    done.current = onDone;
  }, [onDone]);

  useEffect(() => {
    const end = new Date(target).getTime();
    const tick = () => {
      const remaining = end - Date.now();
      setLeft(Math.max(0, remaining));
      if (remaining <= 0) {
        clearInterval(timer);
        done.current();
      }
    };
    const frame = requestAnimationFrame(tick);
    const timer = setInterval(tick, 1000);
    return () => {
      cancelAnimationFrame(frame);
      clearInterval(timer);
    };
  }, [target]);

  const time = left === null ? null : parts(left);
  const cells = [
    { label: "days", value: time?.days },
    { label: "hours", value: time?.hours },
    { label: "minutes", value: time?.minutes },
    { label: "seconds", value: time?.seconds },
  ].filter((cell) => cell.label !== "days" || !time || time.days > 0);

  return (
    <div
      className="countdown"
      role="timer"
      aria-label={time ? `${time.days} days, ${time.hours} hours, and ${time.minutes} minutes to go` : "Time until the event"}
    >
      {cells.map((cell) => (
        <div className="countdown-cell" key={cell.label} aria-hidden="true">
          <span className="countdown-value">
            {cell.value === undefined ? "–" : String(cell.value).padStart(2, "0")}
          </span>
          <span className="countdown-label">{cell.label}</span>
        </div>
      ))}
    </div>
  );
}

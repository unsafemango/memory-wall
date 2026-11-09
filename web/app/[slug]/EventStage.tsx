"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { API_URL } from "@/lib/api";
import { formatEventWhen } from "@/lib/format";
import { occasionFor } from "@/lib/occasions";
import type { EventInfo, Memory } from "@/lib/types";
import { Confetti } from "./Confetti";
import { Countdown } from "./Countdown";
import { LiveWall } from "./LiveWall";

// How often, and how many times, to ask for the memories once the countdown
// ends. The server decides when they open, and its clock can be a little behind.
const RETRY_MS = 3000;
const MAX_RETRIES = 40;

type Props = {
  event: EventInfo;
  memories: Memory[];
  // The memories are hidden until the event.
  locked: boolean;
  // The event is still ahead, as of when the page was made.
  upcoming: boolean;
};

// EventStage is everything on the event page: the confetti, the countdown, and
// the memories, including the moment they open while the page is being watched.
export function EventStage({ event: initialEvent, memories: initialMemories, locked: startLocked, upcoming }: Props) {
  const [event, setEvent] = useState(initialEvent);
  const [memories, setMemories] = useState(initialMemories);
  const [locked, setLocked] = useState(startLocked);
  const [arrived, setArrived] = useState(false);
  const retry = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const occasion = occasionFor(event.occasion);

  useEffect(() => () => clearTimeout(retry.current), []);

  const slug = initialEvent.slug;
  const handleArrived = useCallback(() => {
    setArrived(true);
    if (!startLocked) return;

    let attempts = 0;
    const tryOpen = () => {
      const again = () => {
        attempts += 1;
        if (attempts < MAX_RETRIES) retry.current = setTimeout(tryOpen, RETRY_MS);
      };
      fetch(`${API_URL}/api/events/${slug}/memories`, { cache: "no-store" })
        .then(async (res) => {
          if (!res.ok) return again();
          const data: { event: EventInfo; memories: Memory[]; locked?: boolean } = await res.json();
          if (data.locked) return again();
          setEvent(data.event);
          setMemories(data.memories);
          setLocked(false);
        })
        .catch(again);
    };
    tryOpen();
  }, [startLocked, slug]);

  const showCountdown = upcoming && occasion.celebration && event.event_at;

  return (
    <>
      {occasion.effect && <Confetti effect={occasion.effect} />}
      {arrived && !locked && occasion.effect && <Confetti effect={occasion.effect} burst />}

      <section className="hero">
        {(event.event_date || occasion.celebration) && (
          <span className="hero-eyebrow">{formatEventWhen(event) || "A very special day"}</span>
        )}
        <h1 className="name">{event.title}</h1>
        {!locked && !arrived && <p className="hero-sub">{occasion.heroLine}</p>}
        {showCountdown && !arrived && <Countdown target={event.event_at!} onDone={handleArrived} />}
        {arrived && (
          <p className="hero-arrived" role="status">
            {locked ? "Opening the wall…" : event.event_time ? "It's time!" : "The day is here!"}
          </p>
        )}
      </section>

      {locked ? (
        !arrived && (
          <section className="reveal-card">
            <p className="reveal-title">Something special is waiting</p>
            <p className="reveal-body">
              The memories open when the countdown ends. Come back then, or leave this page open and watch it happen.
            </p>
          </section>
        )
      ) : (
        <LiveWall slug={event.slug} initial={memories} revealed={startLocked} />
      )}

      {!locked && event.closing_message && (
        <section className="closing-wrap">
          <div className="message-wrap">
            <div className="tag">For {event.title}</div>
            <p>{event.closing_message}</p>
            <div className="signoff">{occasion.signoff}</div>
          </div>
        </section>
      )}
    </>
  );
}

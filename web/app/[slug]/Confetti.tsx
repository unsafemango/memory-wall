"use client";
import { useEffect, useState } from "react";
import { DEFAULT_EFFECT, type ConfettiEffect, type ConfettiShape } from "@/lib/occasions";

type Piece = {
  left: number;
  color: string;
  shape: ConfettiShape;
  duration: number;
  delay: number;
  drift: number;
  spin: number;
  scale: number;
};

function makePieces(effect: ConfettiEffect, count: number, burst: boolean): Piece[] {
  const pick = <T,>(list: T[]) => list[Math.floor(Math.random() * list.length)];
  return Array.from({ length: count }, () => {
    const duration = burst ? 2.5 + Math.random() * 2 : (8 + Math.random() * 7) / effect.speed;
    return {
      left: Math.random() * 100,
      color: pick(effect.colors),
      shape: pick(effect.shapes),
      duration,
      // A burst all starts near the top. Otherwise a negative delay means the
      // page is already full of confetti when it opens, not just starting.
      delay: burst ? Math.random() * 0.8 : -Math.random() * duration,
      drift: (Math.random() - 0.5) * 120,
      spin: (Math.random() < 0.5 ? -1 : 1) * (180 + Math.random() * 360),
      scale: 0.8 + Math.random() * 0.5,
    };
  });
}

// Confetti falls behind the page's content, over the whole window. With burst
// set it falls once and quickly instead, for a moment like the countdown ending.
export function Confetti({ effect = DEFAULT_EFFECT, burst = false }: { effect?: ConfettiEffect; burst?: boolean }) {
  const [pieces, setPieces] = useState<Piece[]>([]);

  // Random positions are made after mount. Making them during render would
  // give the server and the browser different HTML and a hydration error.
  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      // Fewer on a phone, where the same number looks much busier.
      const count = Math.round(effect.count * (window.innerWidth < 640 ? 0.6 : 1) * (burst ? 2.5 : 1));
      setPieces(makePieces(effect, count, burst));
    });
    return () => cancelAnimationFrame(frame);
  }, [effect, burst]);

  return (
    <div className={burst ? "fx-layer burst" : "fx-layer"} aria-hidden="true">
      {pieces.map((p, i) => (
        <span
          key={i}
          className={`confetti ${p.shape}`}
          style={
            {
              left: `${p.left}%`,
              background: p.color,
              animationDuration: `${p.duration}s`,
              animationDelay: `${p.delay}s`,
              "--drift": `${p.drift}px`,
              "--spin": `${p.spin}deg`,
              "--scale": p.scale,
            } as React.CSSProperties
          }
        />
      ))}
    </div>
  );
}

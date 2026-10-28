"use client";
import { useEffect, useState } from "react";

const COLORS = ["#ff5da2", "#8b5cf6", "#ffb347", "#ffd93d", "#5ec8f0"];

type Piece = { left: number; color: string; duration: number; delay: number; round: boolean };

export function Confetti({ count = 16 }: { count?: number }) {
  const [pieces, setPieces] = useState<Piece[]>([]);

  // Random positions are made after mount. Making them during render would
  // give the server and the browser different HTML and a hydration error.
  useEffect(() => {
    const frame = requestAnimationFrame(() =>
      setPieces(
        Array.from({ length: count }, () => ({
          left: Math.random() * 96 + 2,
          color: COLORS[Math.floor(Math.random() * COLORS.length)],
          duration: 2.5 + Math.random() * 2.5,
          delay: Math.random() * 3,
          round: Math.random() < 0.3,
        })),
      ),
    );
    return () => cancelAnimationFrame(frame);
  }, [count]);

  return (
    <div className="fx-layer" aria-hidden="true">
      {pieces.map((p, i) => (
        <span
          key={i}
          className={p.round ? "confetti round" : "confetti"}
          style={{
            left: `${p.left}%`,
            background: p.color,
            animationDuration: `${p.duration}s`,
            animationDelay: `${p.delay}s`,
          }}
        />
      ))}
    </div>
  );
}

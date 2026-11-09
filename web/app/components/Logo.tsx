// Logo is the Memory Wall mark: a polaroid with a heart, overlapping a note.
// app/icon.svg is the same drawing, used as the browser tab icon.
export function Logo({ size = 28 }: { size?: number }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width={size} height={size} aria-hidden="true">
      <defs>
        <linearGradient id="mw-logo-bg" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#ff5da2"/>
          <stop offset="1" stopColor="#8b5cf6"/>
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx="15" fill="url(#mw-logo-bg)"/>
      <rect x="11" y="15" width="28" height="33" rx="3" fill="#fff" fillOpacity=".45" transform="rotate(-14 25 31)"/>
      <g transform="rotate(8 37 33)">
        <rect x="20" y="12" width="33" height="40" rx="3" fill="#fff"/>
        <rect x="24" y="16" width="25" height="24" rx="1.5" fill="#fde7f3"/>
        <path d="M36.5 35.5c-5.6-3.9-8.5-6.8-8.5-10.200a4.700 4.700 0 0 1 8.500-2.800a4.700 4.700 0 0 1 8.500 2.800c0 3.400-2.900 6.300-8.500 10.200z" fill="#e63888"/>
      </g>
    </svg>
  );
}

import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // In production the web app and API are on different sites (for example
  // yourapp.vercel.app and yourapi.fly.dev), and browsers won't send the
  // SameSite=Lax session cookie between them. Proxying /api/* through Next.js
  // keeps the browser on one origin. To use it, set API_URL to the API's
  // address and leave NEXT_PUBLIC_API_URL unset. API_URL must be set at build
  // time, since Next.js reads rewrites when it builds.
  async rewrites() {
    if (!process.env.API_URL) return [];
    return [{ source: "/api/:path*", destination: `${process.env.API_URL}/api/:path*` }];
  },
};

export default nextConfig;

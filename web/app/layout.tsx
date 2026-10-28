import type { Metadata } from "next";
import { Fredoka, Quicksand } from "next/font/google";
import "./globals.css";

const fredoka = Fredoka({ subsets: ["latin"], variable: "--font-fredoka" });
const quicksand = Quicksand({ subsets: ["latin"], variable: "--font-quicksand" });

export const metadata: Metadata = {
  title: "Memory Wall",
  description: "Collect notes, photos, and videos from everyone for a birthday, wedding, or any celebration.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${fredoka.variable} ${quicksand.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col">{children}</body>
    </html>
  );
}

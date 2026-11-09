"use client";
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Organizer } from "@/lib/types";
import { Logo } from "./Logo";

// The account page sends this after a save, so the header shows the new name.
export const ORGANIZER_UPDATED = "memorywall:organizer-updated";

function initials(name: string) {
  const parts = name.trim().split(/\s+/);
  const letters = parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : parts[0].slice(0, 2);
  return letters.toUpperCase();
}

function AccountMenu() {
  const router = useRouter();
  const [organizer, setOrganizer] = useState<Organizer | null>(null);
  const [open, setOpen] = useState(false);
  const menu = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const load = () => {
      api("/api/organizer/me")
        .then(async (res) => {
          if (res.ok) setOrganizer(await res.json());
        })
        .catch(() => {});
    };
    load();
    window.addEventListener(ORGANIZER_UPDATED, load);
    return () => window.removeEventListener(ORGANIZER_UPDATED, load);
  }, []);

  // Close when clicking elsewhere or pressing Escape.
  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!menu.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  async function logout() {
    await api("/api/auth/logout", { method: "POST" }).catch(() => {});
    router.push("/login");
  }

  return (
    <div className="account-menu" ref={menu}>
      <button
        type="button"
        className="avatar-button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Account menu"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="avatar">{organizer ? initials(organizer.display_name) : ""}</span>
      </button>
      {open && (
        <div className="account-dropdown" role="menu">
          {organizer && (
            <div className="account-who">
              <span className="field-name">{organizer.display_name}</span>
              <span className="field-hint account-email">{organizer.email}</span>
            </div>
          )}
          <Link role="menuitem" href="/dashboard" className="menu-item" onClick={() => setOpen(false)}>
            Your events
          </Link>
          <Link role="menuitem" href="/dashboard/account" className="menu-item" onClick={() => setOpen(false)}>
            Account settings
          </Link>
          <button role="menuitem" type="button" className="menu-item" onClick={logout}>
            Log out
          </button>
        </div>
      )}
    </div>
  );
}

// AppHeader is the top bar on organizer pages. Logged-in pages get navigation
// and the account menu; login and signup link to each other instead.
export function AppHeader({ mode }: { mode: "signed-in" | "login" | "signup" }) {
  const pathname = usePathname();
  const onEvents = pathname === "/dashboard" || pathname.startsWith("/dashboard/events");

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <Link href={mode === "signed-in" ? "/dashboard" : "/"} className="brand">
          <Logo />
          <span>Memory Wall</span>
        </Link>
        {mode === "signed-in" ? (
          <nav className="app-nav" aria-label="Main">
            <Link
              href="/dashboard"
              className={onEvents ? "nav-link active" : "nav-link"}
              aria-current={onEvents ? "page" : undefined}
            >
              Your events
            </Link>
            <Link href="/dashboard/new" className="btn btn-primary btn-small nav-new">
              <span aria-hidden="true">+</span> New event
            </Link>
            <AccountMenu />
          </nav>
        ) : (
          <nav className="app-nav" aria-label="Account">
            {mode === "login" && <Link href="/signup" className="btn btn-secondary btn-small">Create an account</Link>}
            {mode === "signup" && <Link href="/login" className="btn btn-secondary btn-small">Log in</Link>}
          </nav>
        )}
      </div>
    </header>
  );
}

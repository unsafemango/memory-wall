"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

// AppHeader is the top bar on organizer pages. Logged-in pages get a link
// back to the dashboard and a log out button; login and signup link to each
// other instead.
export function AppHeader({ mode }: { mode: "signed-in" | "login" | "signup" }) {
  const router = useRouter();

  async function logout() {
    await api("/api/auth/logout", { method: "POST" }).catch(() => {});
    router.push("/login");
  }

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <Link href={mode === "signed-in" ? "/dashboard" : "/"} className="brand">
          Memory Wall
        </Link>
        <nav className="app-nav" aria-label="Account">
          {mode === "signed-in" && (
            <>
              <Link href="/dashboard" className="btn btn-ghost btn-small">Your events</Link>
              <button type="button" onClick={logout} className="btn btn-secondary btn-small">Log out</button>
            </>
          )}
          {mode === "login" && <Link href="/signup" className="btn btn-secondary btn-small">Create an account</Link>}
          {mode === "signup" && <Link href="/login" className="btn btn-secondary btn-small">Log in</Link>}
        </nav>
      </div>
    </header>
  );
}

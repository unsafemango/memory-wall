"use client";
import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, errorMessage } from "@/lib/api";
import { AppHeader } from "./components/AppHeader";

// AuthForm is the login and signup form; signup adds a display name.
export function AuthForm({ mode }: { mode: "login" | "signup" }) {
  const router = useRouter();
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const signup = mode === "signup";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const body = signup ? { display_name: displayName, email, password } : { email, password };
      const res = await api(`/api/auth/${mode}`, { method: "POST", body: JSON.stringify(body) });
      if (res.ok) {
        router.push("/dashboard");
        return;
      }
      setError(await errorMessage(res));
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    }
    setSubmitting(false);
  }

  return (
    <div className="app">
      <AppHeader mode={mode} />
      <main className="app-main narrow">
        <div className="panel">
          <h1 className="page-title">{signup ? "Start a memory wall" : "Welcome back"}</h1>
          <p className="page-sub">
            {signup
              ? "Create an account to set up a wall and invite people to it."
              : "Log in to see your events and new memories."}
          </p>
          <form onSubmit={handleSubmit} className="form" style={{ marginTop: 20 }}>
            {signup && (
              <label className="field">
                <span className="field-name">Your name</span>
                <input
                  className="input"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  required
                  maxLength={100}
                  autoComplete="name"
                />
              </label>
            )}
            <label className="field">
              <span className="field-name">Email</span>
              <input
                className="input"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoComplete="email"
              />
            </label>
            <div className="field">
              <label className="field-name" htmlFor="password">Password</label>
              <div className="input-group">
                <input
                  id="password"
                  className="input"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={signup ? 10 : undefined}
                  autoComplete={signup ? "new-password" : "current-password"}
                  aria-describedby={signup ? "password-hint" : undefined}
                />
                <button
                  type="button"
                  className="input-addon"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-pressed={showPassword}
                >
                  {showPassword ? "Hide" : "Show"}
                </button>
              </div>
              {signup && (
                <span className="field-hint" id="password-hint">At least 10 characters.</span>
              )}
            </div>
            {error && <p className="form-error" role="alert">{error}</p>}
            <button type="submit" disabled={submitting} className="btn btn-primary btn-block">
              {submitting ? (signup ? "Creating your account…" : "Logging in…") : signup ? "Create account" : "Log in"}
            </button>
          </form>
        </div>
        <p className="page-sub" style={{ textAlign: "center", marginTop: 16 }}>
          {signup ? (
            <>Already have an account? <Link className="wall-link" href="/login">Log in</Link></>
          ) : (
            <>New here? <Link className="wall-link" href="/signup">Create an account</Link></>
          )}
        </p>
      </main>
    </div>
  );
}

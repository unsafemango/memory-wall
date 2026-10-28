"use client";
import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, errorMessage } from "@/lib/api";
import { buttonClass, inputClass } from "@/lib/ui";

// AuthForm is the login and signup form; signup adds a display name.
export function AuthForm({ mode }: { mode: "login" | "signup" }) {
  const router = useRouter();
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const body = mode === "signup" ? { display_name: displayName, email, password } : { email, password };
      const res = await api(`/api/auth/${mode}`, { method: "POST", body: JSON.stringify(body) });
      if (res.ok) router.push("/dashboard");
      else setError(await errorMessage(res));
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="mx-auto w-full max-w-sm p-6">
      <h1 className="mb-4 text-2xl font-semibold">{mode === "signup" ? "Create your account" : "Log in"}</h1>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        {mode === "signup" && (
          <label className="flex flex-col gap-1">
            Your name
            <input className={inputClass} value={displayName} onChange={(e) => setDisplayName(e.target.value)} required />
          </label>
        )}
        <label className="flex flex-col gap-1">
          Email
          <input className={inputClass} type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" />
        </label>
        <label className="flex flex-col gap-1">
          Password
          <input
            className={inputClass}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={mode === "signup" ? 10 : undefined}
            autoComplete={mode === "signup" ? "new-password" : "current-password"}
          />
          {mode === "signup" && <span className="text-sm opacity-70">At least 10 characters.</span>}
        </label>
        {error && <p className="text-red-600">{error}</p>}
        <button type="submit" disabled={submitting} className={buttonClass}>
          {mode === "signup" ? "Sign up" : "Log in"}
        </button>
      </form>
      <p className="mt-4 text-sm">
        {mode === "signup" ? (
          <>Already have an account? <Link className="underline" href="/login">Log in</Link></>
        ) : (
          <>New here? <Link className="underline" href="/signup">Create an account</Link></>
        )}
      </p>
    </main>
  );
}

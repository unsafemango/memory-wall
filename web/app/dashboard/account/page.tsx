"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, errorMessage } from "@/lib/api";
import type { Organizer } from "@/lib/types";
import { AppHeader, ORGANIZER_UPDATED } from "../../components/AppHeader";
import { Toast, useToast } from "../../components/Toast";

function ProfileForm({ organizer, onSaved }: { organizer: Organizer; onSaved: (o: Organizer) => void }) {
  const [displayName, setDisplayName] = useState(organizer.display_name);
  const [email, setEmail] = useState(organizer.email);
  const [currentPassword, setCurrentPassword] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const emailChanged = email.trim().toLowerCase() !== organizer.email;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const res = await api("/api/organizer/me", {
        method: "PATCH",
        body: JSON.stringify({ display_name: displayName, email, current_password: currentPassword }),
      });
      if (!res.ok) {
        setError(await errorMessage(res));
        return;
      }
      setCurrentPassword("");
      onSaved(await res.json());
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="form">
      <label className="field">
        <span className="field-name">Name</span>
        <input className="input" value={displayName} onChange={(e) => setDisplayName(e.target.value)} required maxLength={100} autoComplete="name" />
      </label>
      <label className="field">
        <span className="field-name">Email</span>
        <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" />
        <span className="field-hint">You use this to log in.</span>
      </label>
      {emailChanged && (
        <label className="field">
          <span className="field-name">Current password</span>
          <input
            className="input"
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            required
            autoComplete="current-password"
          />
          <span className="field-hint">Needed to change the email you log in with.</span>
        </label>
      )}
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="row">
        <button type="submit" disabled={saving} className="btn btn-primary">{saving ? "Saving…" : "Save changes"}</button>
      </div>
    </form>
  );
}

function PasswordForm({ onSaved }: { onSaved: () => void }) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [show, setShow] = useState(false);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const res = await api("/api/organizer/me/password", {
        method: "POST",
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
      });
      if (!res.ok) {
        setError(await errorMessage(res));
        return;
      }
      setCurrentPassword("");
      setNewPassword("");
      onSaved();
    } catch {
      setError("We couldn't reach the server. Try again in a minute.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="form">
      <label className="field">
        <span className="field-name">Current password</span>
        <input
          className="input"
          type="password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          required
          autoComplete="current-password"
        />
      </label>
      <div className="field">
        <label className="field-name" htmlFor="new-password">New password</label>
        <div className="input-group">
          <input
            id="new-password"
            className="input"
            type={show ? "text" : "password"}
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
            minLength={10}
            autoComplete="new-password"
            aria-describedby="new-password-hint"
          />
          <button type="button" className="input-addon" onClick={() => setShow((v) => !v)} aria-pressed={show}>
            {show ? "Hide" : "Show"}
          </button>
        </div>
        <span className="field-hint" id="new-password-hint">
          At least 10 characters. Your other devices will be logged out.
        </span>
      </div>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="row">
        <button type="submit" disabled={saving} className="btn btn-secondary">{saving ? "Changing…" : "Change password"}</button>
      </div>
    </form>
  );
}

export default function AccountPage() {
  const router = useRouter();
  const toast = useToast();
  const [organizer, setOrganizer] = useState<Organizer | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api("/api/organizer/me")
      .then(async (res) => {
        if (res.status === 401) router.push("/login");
        else if (res.ok) setOrganizer(await res.json());
        else setError("We couldn't load your account.");
      })
      .catch(() => setError("We couldn't reach the server. Try again in a minute."));
  }, [router]);

  return (
    <div className="app">
      <AppHeader mode="signed-in" />
      <main className="app-main form-page">
        <h1 className="page-title">Account settings</h1>
        {error && <div className="panel empty" style={{ marginTop: 20 }}><p>{error}</p></div>}
        {!error && !organizer && <p className="page-sub" aria-busy="true">Loading…</p>}
        {organizer && (
          <p className="page-sub">
            Member since {new Date(organizer.created_at).toLocaleDateString(undefined, { month: "long", year: "numeric" })}
          </p>
        )}
        {organizer && (
          <div className="stack" style={{ marginTop: 20 }}>
            <section className="panel">
              <h2 className="section-title">Profile</h2>
              <p className="section-sub">Your name is shown in your dashboard.</p>
              <ProfileForm
                key={organizer.email + organizer.display_name}
                organizer={organizer}
                onSaved={(updated) => {
                  setOrganizer(updated);
                  window.dispatchEvent(new Event(ORGANIZER_UPDATED));
                  toast.show("Profile saved");
                }}
              />
            </section>
            <section className="panel">
              <h2 className="section-title">Password</h2>
              <p className="section-sub">Choose a new password for logging in.</p>
              <PasswordForm onSaved={() => toast.show("Password changed")} />
            </section>
          </div>
        )}
      </main>
      <Toast message={toast.message} />
    </div>
  );
}

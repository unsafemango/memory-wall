"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppHeader } from "../../components/AppHeader";
import { EventForm } from "../EventForm";

export default function NewEventPage() {
  const router = useRouter();

  return (
    <div className="app">
      <AppHeader mode="signed-in" />
      <main className="app-main form-page">
        <Link href="/dashboard" className="back-link">← Your events</Link>
        <section className="panel">
          <h1 className="page-title">New event</h1>
          <p className="section-sub" style={{ marginTop: 6 }}>You can change any of this later.</p>
          <EventForm
            onSaved={(_, id) => router.push(id ? `/dashboard/events/${id}` : "/dashboard")}
            onCancel={() => router.push("/dashboard")}
          />
        </section>
      </main>
    </div>
  );
}

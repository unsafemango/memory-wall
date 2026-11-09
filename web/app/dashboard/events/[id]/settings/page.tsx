"use client";
import { EventForm } from "../../../EventForm";
import { useManage } from "../context";

export default function EventSettingsPage() {
  const { event, reload, toast } = useManage();

  return (
    <>
      <div className="content-head">
        <h1 className="page-title">Event settings</h1>
        <p className="page-sub">Change the details of this event.</p>
      </div>
      <section className="panel">
        <EventForm
          key={JSON.stringify(event)}
          event={event}
          onSaved={() => {
            toast("Settings saved");
            reload();
          }}
        />
      </section>
    </>
  );
}

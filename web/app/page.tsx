"use client";
import { useEffect, useState } from "react";

export default function Home() {
  const [status, setStatus] = useState("checking...");

  useEffect(() => {
    fetch("http://localhost:8080/api/healthz")
      .then((res) => res.json())
      .then((data) => setStatus(data.status))
      .catch(() => setStatus("unreachable"));
  }, []);

  return <main style={{ padding: 40 }}>API status: {status}</main>;
}

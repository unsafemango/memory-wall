import { useSyncExternalStore } from "react";

const subscribe = () => () => {};

// useOrigin returns the site's origin (http://localhost:3000) in the browser
// and an empty string during server rendering, without a hydration mismatch.
export function useOrigin() {
  return useSyncExternalStore(subscribe, () => window.location.origin, () => "");
}

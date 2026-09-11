import { useEffect, useRef } from "react";

/**
 * Returns a ref to attach to an element; `onOutside` fires on any document
 * click landing outside it. Four dropdowns had hand-rolled byte-identical
 * copies of this effect (AppShell twice, the Dropdown kit, and the admin role
 * picker).
 *
 * The listener subscribes once on mount and reads the callback through a ref,
 * which is what the copies did by closing over a `setState` identity React
 * keeps stable. Depending on `onOutside` directly would instead resubscribe on
 * every render, because every caller passes an inline arrow.
 */
export function useOutsideClick<T extends HTMLElement>(onOutside: () => void) {
  const ref = useRef<T>(null);
  const latest = useRef(onOutside);

  useEffect(() => {
    latest.current = onOutside;
  });

  useEffect(() => {
    const handle = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        latest.current();
      }
    };
    document.addEventListener("click", handle);
    return () => document.removeEventListener("click", handle);
  }, []);

  return ref;
}

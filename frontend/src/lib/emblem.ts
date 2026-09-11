import type React from "react";

/**
 * Background style for a character's emblem banner, or `undefined` when the
 * character has no resolvable emblem URL — returning `undefined` rather than an
 * empty object lets the caller fall back to the CSS-defined placeholder instead
 * of painting an empty image over it.
 */
export function emblemStyle(url?: string): React.CSSProperties | undefined {
  return url
    ? {
        backgroundImage: `url(${url})`,
        backgroundSize: "cover",
        backgroundPosition: "center",
      }
    : undefined;
}

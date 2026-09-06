import {
  BROWSER_SESSION_STORAGE_KEY,
  browserSessionClient,
} from "../lib/browserSessionBrowser";
import type { APIGuardianTrackerUser } from "../types/api";
import { sampleUser } from "./testServer";

/** Seed an atomic projection and deliver the event a second browser document receives. */
export function seedBrowserSession(
  user: APIGuardianTrackerUser = sampleUser,
  accessToken = "test-token",
) {
  // Initialize the test singleton before publishing a fixture revision.
  browserSessionClient.getSnapshot();
  const current = JSON.parse(
    localStorage.getItem(BROWSER_SESSION_STORAGE_KEY) ?? "{}",
  ) as { revision?: string };
  const raw = JSON.stringify({
    schemaVersion: 1,
    revision: (BigInt(current.revision ?? "1") + 1n).toString(),
    lineage: crypto.randomUUID(),
    projection: { status: "authenticated", user, accessToken },
  });
  localStorage.setItem(BROWSER_SESSION_STORAGE_KEY, raw);
  window.dispatchEvent(
    new StorageEvent("storage", {
      key: BROWSER_SESSION_STORAGE_KEY,
      newValue: raw,
    }),
  );
}
